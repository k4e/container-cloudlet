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

public class DeployTest {
    
    private final UUID seshId;

    public DeployTest(UUID seshId) {
        this.seshId = seshId;
    }

    public void exec(String host, Request.Deploy.Type type, String srcAddr)
    throws IOException, InterruptedException {
        if (type == null) {
            throw new IllegalArgumentException("type == null");
        }
        if ((type == Request.Deploy.Type.FWD || type == Request.Deploy.Type.LM)
                && Strings.isNullOrEmpty(srcAddr)) {
            throw new IllegalArgumentException("type is FWD|LM but srcAddr is empty");
        }
        Gson gson = new Gson();
        Request req = CloudletClient.createAppSampleRequest(type, srcAddr);
        String reqStr = gson.toJson(req);
        ProtocolHeader header = ProtocolHeader.create(seshId);
        byte headerBytes[] = header.getBytes();
        char cbuf[] = new char[4096];
        byte buf[] = new byte[1024];
        byte testData[] = "Hello_world ABCD".getBytes();
        System.out.println("--- Deploy test start ---");
        long start = System.currentTimeMillis();
        long end;
        System.out.println("Send deploy request to Cloudlet Controller");
        try (Socket sockAPI = new Socket(host, CloudletClient.DEFAULT_CLOUDLET_PORT)) {
            PrintWriter writer = new PrintWriter(sockAPI.getOutputStream());
            writer.println(reqStr);
            writer.flush();
            System.out.printf("Sent: %s\n", reqStr);
            InputStreamReader reader = new InputStreamReader(sockAPI.getInputStream());
            int ccCnt = reader.read(cbuf);
            System.out.printf("Recv: %s\n", ccCnt > 0 ? new String(cbuf, 0, ccCnt) : "(none)");
        }
        System.out.print("Connect to Session Service");
        while (true) {
            System.out.print(".");
            int appReadCnt = 0;
            boolean connReset = false;
            try (Socket sockApp = new Socket(host, CloudletClient.DEFAULT_APP_EXT_PORT)) {
                sockApp.getOutputStream().write(headerBytes);
                sockApp.getOutputStream().flush();
                sockApp.getOutputStream().write(testData);
                appReadCnt = sockApp.getInputStream().read(buf);
            } catch (SocketException e) {
                if (e.getMessage().contains("Connection reset")) {
                    connReset = true;
                } else {
                    throw e;
                }
            }
            if (connReset || appReadCnt < 0) {
                Thread.sleep(100);
                continue;
            } else if (0 < appReadCnt) {
                byte readData[] = Arrays.copyOfRange(buf, 0, appReadCnt);
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
        end = System.currentTimeMillis();
        System.out.println("--- Creation test end ---");
        System.out.println("Time elapsed (ms): " + (end - start));
        System.out.printf("Please clean up the server: run with args: remove %s\n", host);
    }
}
