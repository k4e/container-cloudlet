package com.github.k4e;

import java.io.IOException;
import java.io.OutputStream;
import java.net.Socket;
import java.util.Random;
import java.util.UUID;

import com.github.k4e.types.ProtocolHeader;

public class Experiment {

    private static final int DATA_SIZE = 1024;

    private final Random random = new Random(101L);
    private final UUID seshId;

    public Experiment(UUID seshId) {
        this.seshId = seshId;
    }

    public void exec(String hostA, int portA, String hostB, int portB)
    throws IOException {
        byte[] data = generateBytes(DATA_SIZE);
        byte[] buf = new byte[DATA_SIZE * 4];
        long start = System.currentTimeMillis();
        System.out.println("# --> Server A");
        try (Socket sockA = new Socket(hostA, portA)) {
            sendHeader(hostA, portA, null, Integer.valueOf(0).shortValue(), sockA);
            for (int i = 0; i < 4; ++i) {
                sockA.getOutputStream().write(data);
                sockA.getOutputStream().flush();
                sockA.getInputStream().read(buf);
                long end = System.currentTimeMillis();
                System.out.println(end - start);
                start = end;
            }
        }
        System.out.println("# --> Server B");
        try (Socket sockB = new Socket(hostB, portB)) {
            sendHeader(hostB, portB, hostA, Integer.valueOf(portA).shortValue(), sockB);
            for (int i = 0; i < 4; ++i) {
                sockB.getOutputStream().write(data);
                sockB.getOutputStream().flush();
                sockB.getInputStream().read(buf);
                long end = System.currentTimeMillis();
                System.out.println(end - start);
                start = end;
            }
        }
    }

    private byte[] generateBytes(int b) {
        byte[] a = new byte[b];
        for (int i = 0; i < b; ++i) {
            a[i] = Integer.valueOf('a' + random.nextInt(26)).byteValue();
        }
        return a;
    }

    private void sendHeader(String host, int port, String fwdHostIp, short fwdHostPort,
            Socket sock) throws IOException {
        ProtocolHeader header = ProtocolHeader.create(seshId, fwdHostIp, fwdHostPort, false);
        OutputStream out = sock.getOutputStream();
        byte[] b = header.getBytes();
        out.write(b);
        out.flush();
    }
}
