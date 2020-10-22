package com.github.k4e.exp;

import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.PrintWriter;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.util.Random;
import java.util.UUID;

import com.github.k4e.CloudletClient;
import com.github.k4e.types.ProtocolHeader;
import com.github.k4e.types.Request;
import com.google.common.base.Strings;
import com.google.gson.Gson;

public class SpeedTest {

    public static final int DEFAULT_COUNT = 10;
    private final Random random = new Random(101L);
    private final UUID seshId;

    public SpeedTest(UUID seshId) {
        this.seshId = seshId;
    }

    public void exec(
        String hostAddr,
        Request.Deploy.Type type,
        int dataSizeKB,
        int count,
        String srcAddr,
        boolean noWait,
        boolean fullCheck
    ) throws IOException, InterruptedException {
        if ((type == Request.Deploy.Type.FWD || type == Request.Deploy.Type.LM)
                && Strings.isNullOrEmpty(srcAddr)) {
            throw new IllegalArgumentException("type is FWD|LM but srcAddr is empty");
        }
        final int dataSizeBytes = dataSizeKB * 1024;
        if (count < 0) {
            count = DEFAULT_COUNT;
        }
        Gson gson = new Gson();
        ProtocolHeader header = ProtocolHeader.create(seshId);
        byte headerBytes[] = header.getBytes();
        String req = null;
        if (type != null) {
            Request r = CloudletClient.createAppSampleRequest(type, srcAddr);
            req = gson.toJson(r);
        }
        // byte testData[] = "Hello_world ABCD".getBytes();
        byte[] data = generateBytes(dataSizeBytes);
        byte[] buf = new byte[dataSizeBytes * 4];
        char[] cbuf = new char[4096];
        int consistent = 0;
        int inconsistent = 0;
        System.out.println("--- Speed test start ---");
        long timeline = 0;
        long firstTime = System.nanoTime();
        long startTime;
        long endTime;
        if (type != null) {    
            System.out.println("Send deploy request to Cloudlet Controller");
            try(Socket sockAPI = new Socket(hostAddr, CloudletClient.DEFAULT_CLOUDLET_PORT)) {
                PrintWriter writer = new PrintWriter(sockAPI.getOutputStream());
                writer.println(req);
                writer.flush();
                System.out.printf("Sent: %s\n", req);
                InputStreamReader reader = new InputStreamReader(sockAPI.getInputStream());
                int apiReadCount = reader.read(cbuf);
                System.out.printf("Recv: %s\n", apiReadCount > 0 ? new String(cbuf, 0, apiReadCount) : "(none)");
            }
        }
        // System.out.print("Connect to Session Service");
        // while (true) {
        //     System.out.print(".");
        //     int appReadCnt = 0;
        //     boolean connReset = false;
        //     try (Socket sockApp = new Socket(hostAddr, CloudletClient.DEFAULT_APP_EXT_PORT)) {
        //         OutputStream out = sockApp.getOutputStream();
        //         InputStream in = sockApp.getInputStream();
        //         out.write(headerBytes);
        //         out.flush();
        //         out.write(testData);
        //         out.flush();
        //         appReadCnt = in.read(buf);
        //     } catch (SocketException e) {
        //         if (e.getMessage().contains("Connection reset")) {
        //             connReset = true;
        //         } else {
        //             throw e;
        //         }
        //     }
        //     if (connReset || appReadCnt < 0) {
        //         Thread.sleep(100);
        //         continue;
        //     } else if (appReadCnt > 0) {
        //         byte readData[] = Arrays.copyOfRange(buf, 0, appReadCnt);
        //         String testDataStr = new String(testData);
        //         String readDataStr = new String(readData);
        //         System.out.println();
        //         if (Arrays.equals(testData, readData)) {
        //             System.out.printf("OK: wrote: %s, read: %s\n", testDataStr, readDataStr);
        //         } else {
        //             System.out.printf("FAIL: wrote: %s, read: %s\n", testDataStr, readDataStr);
        //         }
        //         break;
        //     }
        // }
        endTime = System.nanoTime();
        System.out.println("CreationTime (ms): " + (endTime - firstTime) / 1000000);
        System.out.println("Proceed to throughput test");
        Socket sockApp = new Socket();
        try {
            sockApp.connect(new InetSocketAddress(hostAddr, CloudletClient.DEFAULT_APP_EXT_PORT));
            OutputStream out = sockApp.getOutputStream();
            InputStream in = sockApp.getInputStream();
            out.write(headerBytes);
            out.flush();
            for (int i = 0; i < count; ++i) {
                int wroteSz, readSz;
                startTime = System.nanoTime();
                timeline = startTime - firstTime;
                out.write(data);
                out.flush();
                wroteSz = data.length;
                readSz = 0;
                while (readSz < wroteSz) {
                    int n = in.read(buf, readSz, buf.length - readSz);
                    if (n <= 0) {
                        System.out.println("Read returned " + n);
                        break;
                    }
                    readSz += n;
                }
                endTime = System.nanoTime();
                long timelineMs = timeline / 1000000;
                long elapsed = endTime - startTime;
                long elapsedMs = (endTime - startTime) / 1000000;
                double throughput = ((double)dataSizeKB / ((double)elapsed / 1000000000.));
                System.out.printf("%d\t%d\t%f\n", timelineMs, elapsedMs, throughput);
                boolean sizeTest;
                if (wroteSz == readSz) {
                    sizeTest = true;
                } else {
                    sizeTest = false;
                    System.out.printf("Size test failed: wrote: %dB, read: %dB\n", wroteSz, readSz);
                }
                boolean randomTest = true;
                int upperBound = Math.min(readSz, wroteSz);
                for (int t = 0; t < (fullCheck ? upperBound : Math.min(100, upperBound)); ++t) {
                    int j = (fullCheck ? t : random.nextInt(upperBound));
                    if (data[j] != buf[j]) {
                        randomTest = false;
                        System.out.printf("Random test failed: wrote[%d]=0x%x but read[%d]=0x%x\n",
                                j, data[j], j, buf[j]);
                        break;
                    }
                }
                if (sizeTest && randomTest) {
                    ++consistent;
                } else {
                    ++inconsistent;
                }
            }
        } finally {
            sockApp.close();
        }
        System.out.printf("Test result: consistent: %d, inconsistent: %d, full-check: %s\n",
                    consistent, inconsistent, String.valueOf(fullCheck));
        System.out.printf("May clean up the server: run with args: remove %s\n", hostAddr);
    }

    private byte[] generateBytes(int b) {
        byte[] a = new byte[b];
        for (int i = 0; i < b; ++i) {
            a[i] = Integer.valueOf('a' + random.nextInt(26)).byteValue();
        }
        return a;
    }
}
