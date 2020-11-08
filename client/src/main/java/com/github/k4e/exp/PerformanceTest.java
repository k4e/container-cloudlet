package com.github.k4e.exp;

import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.PrintWriter;
import java.net.Socket;
import java.net.SocketException;
import java.util.Arrays;
import java.util.Random;
import java.util.function.Supplier;

import com.github.k4e.CloudletClient;
import com.github.k4e.types.Request;
import com.google.common.base.Strings;
import com.google.gson.Gson;

public class PerformanceTest {

    public static final int DEFAULT_COUNT = 10;
    public static final int DEFAULT_DURATION_SEC = 20;
    public static final long DEFAULT_TIME_X_INTERVAL = 500L * 1000L * 1000L;
    private final Random random = new Random(101L);

    public void deployTest(String hostAddr, Request.Deploy.Type type, String srcAddr)
    throws IOException, InterruptedException {
        if (type == null) {
            throw new IllegalArgumentException("type == null");
        }
        if ((type == Request.Deploy.Type.FWD || type == Request.Deploy.Type.LM)
                && Strings.isNullOrEmpty(srcAddr)) {
            throw new IllegalArgumentException("type is FWD|LM but srcAddr is empty");
        }
        Gson gson = new Gson();
        Request r = CloudletClient.createAppSampleRequest(type, srcAddr);
        String req = gson.toJson(r);
        byte testData[] = "Hello_world @#$%".getBytes();
        char[] cbuf = new char[4096];
        byte[] buf = new byte[4096];
        System.out.println("--- Deploy test start ---");
        long startTime = System.nanoTime();
        System.out.println("Send deploy request to the Cloudlet API");
        try(Socket sockAPI = new Socket(hostAddr, CloudletClient.DEFAULT_CLOUDLET_PORT)) {
            PrintWriter writer = new PrintWriter(sockAPI.getOutputStream());
            writer.println(req);
            writer.flush();
            System.out.printf("Sent: %s\n", req);
            InputStreamReader reader = new InputStreamReader(sockAPI.getInputStream());
            int apiReadCount = reader.read(cbuf);
            System.out.printf("Recv: %s\n", apiReadCount > 0 ? new String(cbuf, 0, apiReadCount) : "(none)");
        }
        System.out.println("Send test data to the app");
        while (true) {
            int appReadCnt = 0;
            boolean connReset = false;
            try (Socket sockApp = new Socket(hostAddr, CloudletClient.DEFAULT_APP_EXT_PORT)) {
                OutputStream out = sockApp.getOutputStream();
                InputStream in = sockApp.getInputStream();
                out.write(testData);
                out.flush();
                appReadCnt = in.read(buf);
            } catch (SocketException e) {
                if (e.getMessage().contains("Connection reset")) {
                    connReset = true;
                } else {
                    throw e;
                }
            }
            if (connReset || appReadCnt < 0) {
                Thread.sleep(10);
                continue;
            } else if (appReadCnt > 0) {
                byte readData[] = Arrays.copyOfRange(buf, 0, appReadCnt);
                String testDataStr = new String(testData);
                String readDataStr = new String(readData);
                if (Arrays.equals(testData, readData)) {
                    System.out.printf("OK: wrote: %s, read: %s\n", testDataStr, readDataStr);
                } else {
                    System.out.printf("FAIL: wrote: %s, read: %s\n", testDataStr, readDataStr);
                }
                break;
            }
        }
        long endTime = System.nanoTime();
        System.out.println("Deploy time (ms): " + (endTime - startTime) / 1000000);
        System.out.println("--- Deploy test finish ---");
    }

    public void latencyTest(String hostAddr, int dataSizeKB, int count, boolean fullCheck)
    throws IOException {
        final int dataSizeBytes = dataSizeKB * 1024;
        if (count < 0) {
            count = DEFAULT_COUNT;
        }
        byte[] data = generateBytes(dataSizeBytes);
        byte[] buf = new byte[dataSizeBytes];
        int consistent = 0;
        int inconsistent = 0;
        System.out.println("--- Latency test start ---");
        long startTime;
        long endTime;
        try (Socket sockApp = new Socket(hostAddr, CloudletClient.DEFAULT_APP_EXT_PORT)) {
            OutputStream out = sockApp.getOutputStream();
            InputStream in = sockApp.getInputStream();
            for (int i = 0; i < count; ++i) {
                startTime = System.nanoTime();
                out.write(data);
                out.flush();
                int wroteSz = data.length;
                int readSz = 0;
                while (readSz < wroteSz) {
                    int n = in.read(buf, readSz, buf.length - readSz);
                    if (n <= 0) {
                        System.out.println("Read returned " + n);
                        break;
                    }
                    readSz += n;
                }
                endTime = System.nanoTime();
                long elapsed = endTime - startTime;
                System.out.printf("%d\n", elapsed);
                boolean szTest = sizeTest(wroteSz, readSz);
                boolean randTest = randomTest(data, wroteSz, buf, readSz, fullCheck);
                if (szTest && randTest) {
                    ++consistent;
                } else {
                    ++inconsistent;
                }
            }
        }
        System.out.printf("Test result: consistent: %d, inconsistent: %d, full-check: %s\n",
                    consistent, inconsistent, String.valueOf(fullCheck));
        System.out.println("--- Latency test finish ---");
    }

    public void throughputTest(String hostAddr, Request.Deploy.Type type, String srcAddr, int duration)
    throws IOException, InterruptedException {
        if ((type == Request.Deploy.Type.FWD || type == Request.Deploy.Type.LM)
                && Strings.isNullOrEmpty(srcAddr)) {
            throw new IllegalArgumentException("type is FWD|LM but srcAddr is empty");
        }
        if (duration < 0) {
            duration = DEFAULT_DURATION_SEC;
        }
        final int dataSizeBytes = 1 * 1024 * 1024;
        Gson gson = new Gson();
        Request request = CloudletClient.createAppSampleRequest(type, srcAddr);
        String req = gson.toJson(request);
        byte[] data = generateBytes(dataSizeBytes);
        byte[] buf = new byte[dataSizeBytes];
        char[] cbuf = new char[4096];
        int consistent = 0;
        int inconsistent = 0;
        int cntError = 0;
        String lastError = null;
        long nextTimeX = 0L;
        int cntTimeSection = 0;
        System.out.println("--- Throughput test start ---");
        long firstTime = System.nanoTime();
        Supplier<Long> getCurrentTimeX = () -> { return System.nanoTime() - firstTime; };
        System.out.println("Send deploy request to the Cloudlet API");
        try(Socket sockAPI = new Socket(hostAddr, CloudletClient.DEFAULT_CLOUDLET_PORT)) {
            PrintWriter writer = new PrintWriter(sockAPI.getOutputStream());
            writer.println(req);
            writer.flush();
            System.out.printf("Sent: %s\n", req);
            InputStreamReader reader = new InputStreamReader(sockAPI.getInputStream());
            int apiReadCount = reader.read(cbuf);
            System.out.printf("Recv: %s\n", apiReadCount > 0 ? new String(cbuf, 0, apiReadCount) : "(none)");
        }
        System.out.println("Deploy time (ms): " + (System.nanoTime() - firstTime) / 1000000);
        System.out.println("Send test data to the app");
        System.out.println("Time(s)\tThruput(MiB/s)");
        while (getCurrentTimeX.get() < duration * 1000000000L) {
            long accumBytes = 0L;
            try (Socket sockApp = new Socket(hostAddr, CloudletClient.DEFAULT_APP_EXT_PORT)) {
                OutputStream out = sockApp.getOutputStream();
                InputStream in = sockApp.getInputStream();
                while (getCurrentTimeX.get() < duration * 1000000000L) {
                    boolean szTest = true;
                    out.write(data);
                    out.flush();
                    int wroteSz = data.length;
                    int readSz = 0;
                    while (readSz < wroteSz) {
                        int n = in.read(buf, readSz, buf.length - readSz);
                        if (!(n > 0)) {
                            System.out.println("# Read returned " + n);
                            break;
                        }
                        readSz += n;
                        accumBytes += n;
                    }
                    if (!sizeTest(wroteSz, readSz)) {
                        szTest = false;
                    }
                    long currentTimeX = getCurrentTimeX.get();
                    while (nextTimeX <= currentTimeX) {
                        double timeXS = (cntTimeSection * DEFAULT_TIME_X_INTERVAL) / 1000000000.;
                        System.out.printf("%f\t%f\n", timeXS, 0.);
                        nextTimeX += DEFAULT_TIME_X_INTERVAL;
                        ++cntTimeSection;
                    }
                    if (nextTimeX <= currentTimeX + DEFAULT_TIME_X_INTERVAL) {
                        double timeXS = (cntTimeSection * DEFAULT_TIME_X_INTERVAL) / 1000000000.;
                        double dataSizeMB = ((double)(accumBytes)) / (1024 * 1024);
                        double thruput = (dataSizeMB / (DEFAULT_TIME_X_INTERVAL / 1000000000.));
                        System.out.printf("%f\t%f\n", timeXS, thruput);
                        nextTimeX += DEFAULT_TIME_X_INTERVAL;
                        ++cntTimeSection;
                        accumBytes = 0L;
                    }
                    if (szTest) {
                        ++consistent;
                    } else {
                        ++inconsistent;
                    }
                }
            } catch (IOException e) {
                ++cntError;
                System.out.println("# Exception: " + e.getMessage());
                lastError = e.getMessage();
                Thread.sleep(10);
            }
        }
        System.out.printf("Test result: consistent: %d, inconsistent: %d\n", consistent, inconsistent);
        System.out.printf("Error count: %d\n", cntError);
        if (!Strings.isNullOrEmpty(lastError)) {
            System.out.println("Last Error: " + lastError);
        }
        System.out.println("--- Throughput test finish ---");
    }

    private byte[] generateBytes(int b) {
        byte[] a = new byte[b];
        for (int i = 0; i < b; ++i) {
            a[i] = Integer.valueOf('a' + random.nextInt(26)).byteValue();
        }
        return a;
    }

    private boolean sizeTest(int wroteSz, int readSz) {
        boolean ans;
        if (wroteSz == readSz) {
            ans = true;
        } else {
            ans = false;
            System.out.printf("# Size test failed: wrote: %dB, read: %dB\n", wroteSz, readSz);
        }
        return ans;
    }

    private boolean randomTest(byte[] data, int wroteSz, byte[] buf, int readSz, boolean fullCheck) {
        boolean ans = true;
        int upperBound = Math.min(readSz, wroteSz);
        for (int t = 0; t < (fullCheck ? upperBound : Math.min(100, upperBound)); ++t) {
            int j = (fullCheck ? t : random.nextInt(upperBound));
            if (data[j] != buf[j]) {
                ans = false;
                System.out.printf("# Random test failed: wrote[%d]=0x%x but read[%d]=0x%x\n",
                        j, data[j], j, buf[j]);
                break;
            }
        }
        return ans;
    }
}
