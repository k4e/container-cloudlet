package com.github.k4e;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.Socket;
import java.util.Random;
import java.util.UUID;

import com.github.k4e.types.ProtocolHeader;

public class Experiment {

    private final Random random = new Random(101L);
    private final UUID seshId;

    public Experiment(UUID seshId) {
        this.seshId = seshId;
    }

    public void exec(String hostA, int portA, String hostB, int portB, int dataSizeKB)
    throws IOException {
        final int dataSizeBytes = dataSizeKB * 1024;
        byte[] data = generateBytes(dataSizeBytes);
        byte[] buf = new byte[dataSizeBytes * 4];
        System.out.println(dataSizeBytes);
        ProtocolHeader headerA = ProtocolHeader.create(seshId, null, (short)0, false);
        ProtocolHeader headerB = ProtocolHeader.create(seshId, hostA, (short)portA, false);
        System.out.println("# --> Server A");
        routine(hostA, portA, headerA, data, buf);
        System.out.println("# --> Server B");
        routine(hostB, portB, headerB, data, buf);
    }

    private void routine(String host, int port, ProtocolHeader header, byte[] data, byte[] buf)
    throws IOException {
        long start, end;
        int consistent = 0, inconsistent = 0;
        Socket sock = null;
        try {
            start = System.nanoTime();
            sock = new Socket(host, port);
            OutputStream out = sock.getOutputStream();
            InputStream in = sock.getInputStream();
            byte[] headBytes = header.getBytes();
            start = System.nanoTime();
            out.write(headBytes);
            out.flush();
            byte[] pollData = new byte[] {0};
            out.write(pollData);
            in.read(buf);
            end = System.nanoTime();
            System.out.println(end - start);
            for (int i = 0; i < 10; ++i) {
                start = System.nanoTime();
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
                end = System.nanoTime();
                System.out.println(end - start);
                boolean sizeTest;
                if (wroteSz == readSz) {
                    sizeTest = true;
                } else {
                    sizeTest = false;
                    System.out.printf("Size test failed: wrote: %dB, read: %dB\n", wroteSz, readSz);
                }
                boolean randomTest = true;
                int upperBound = Math.min(readSz, wroteSz);
                for (int t = 0; t < Math.min(100, upperBound); ++t) {
                    int j = random.nextInt(upperBound);
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
            System.out.printf("Test result: consistent: %d, inconsistent: %d\n", consistent, inconsistent);
            if (sock != null) {
                sock.close();
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
}
