package com.github.k4e.exp;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;
import java.net.SocketException;
import java.util.Arrays;
import java.util.UUID;

import com.github.k4e.CloudletClient;
import com.github.k4e.types.ProtocolHeader;
import com.github.k4e.types.Request;
import com.google.common.base.Strings;
import com.google.gson.Gson;

public class CreationTest {
    
    private final UUID seshId;

    public CreationTest(UUID seshId) {
        this.seshId = seshId;
    }

    public void exec(
        String host,
        int cloudletPort,
        int sessionPort,
        boolean createApp,
        String fwdHost,
        short fwdPort
    ) throws IOException {
        if (!createApp) {
            if (Strings.isNullOrEmpty(fwdHost) || fwdPort == 0) {
                throw new IllegalArgumentException(String.format(
                    "If createApp == true, it requires fwdHost and fwdPort (given %s:%s)",
                    fwdHost, fwdPort));
            }
        }
        Gson gson = new Gson();
        Request req = CloudletClient.getAppSampleRequest(createApp);
        ProtocolHeader header = ProtocolHeader.create(seshId, fwdHost, fwdPort, false);
        byte headerBytes[] = header.getBytes();
        String msg = gson.toJson(req);
        char cbuf[] = new char[4096];
        byte bbuf[] = new byte[1024];
        byte testData[] = "Hello world ABCD".getBytes();
        Socket sockCC = null;
        try {
            System.out.println("--- Creation test start ---");
            long start = System.currentTimeMillis();
            System.out.println("Sending creation request to Cloudlet Controller");
            sockCC = new Socket(host, cloudletPort);
            PrintWriter writer = new PrintWriter(sockCC.getOutputStream());
            writer.println(msg);
            writer.flush();
            System.out.printf("Sent: %s\n", msg);
            InputStreamReader reader = new InputStreamReader(sockCC.getInputStream());
            int ccCnt = reader.read(cbuf);
            System.out.printf("Recv: %s\n", ccCnt > 0 ? new String(cbuf, 0, ccCnt) : "(none)");
            sockCC.close();
            System.out.println("Connecting to Session Service");
            System.out.print("Pending");
            while (true) {
                System.out.print(".");
                int ssCnt = 0;
                boolean connReset = false;
                try (Socket sockSS = new Socket(host, sessionPort)) {
                    sockSS.getOutputStream().write(headerBytes);
                    sockSS.getOutputStream().flush();
                    sockSS.getOutputStream().write(testData);
                    ssCnt = sockSS.getInputStream().read(bbuf);
                } catch (SocketException e) {
                    if (e.getMessage().contains("Connection reset")) {
                        connReset = true;
                    } else {
                        throw e;
                    }
                }
                if (connReset || ssCnt < 0) {
                    Thread.sleep(100);
                    continue;
                } else if (0 < ssCnt) {
                    byte readData[] = Arrays.copyOfRange(bbuf, 0, ssCnt);
                    String testDataStr = new String(testData);
                    String readDataStr = new String(readData);
                    System.out.println();
                    if (Arrays.equals(testData, readData)) {
                        System.out.printf("OK: wrote: %s, read: %s\n", testDataStr, readDataStr);
                    } else {
                        System.out.printf("FAIL: wrote: %s, read: %s\n", testDataStr, readDataStr);
                    }
                    break;
                }
            }
            long end = System.currentTimeMillis();
            System.out.println("--- Creation test end ---");
            System.out.println("Time elapsed (ms): " + (end - start));
        } catch (InterruptedException e){
            e.printStackTrace();
        } finally {
            if (sockCC != null && !sockCC.isClosed()) {
                sockCC.close();
            }
            System.out.printf("Please clean up the server: run with args: delete %s %d\n",
                    host, cloudletPort);
        }
    }
}
