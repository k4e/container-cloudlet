package com.github.k4e;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.PrintWriter;
import java.net.Socket;
import java.net.UnknownHostException;
import java.util.UUID;

import com.github.k4e.types.ProtocolHeader;

public class SessionClient {

    public static SessionClient of(String host, int port, UUID sessionId,
            String fwdHostIp, short fwdHostPort, boolean resume)
    throws UnknownHostException {
        ProtocolHeader header = ProtocolHeader.create(sessionId, fwdHostIp, fwdHostPort, resume);
        return new SessionClient(host, port, header);
    }

    class SessionProc implements SocketProc {
        public void accept(Socket sock) throws IOException {
            Thread receiver = new Thread(() -> {
                try {
                    char[] buf = new char[8192];
                    InputStreamReader reader = new InputStreamReader(sock.getInputStream());
                    while (true) {
                        while (!reader.ready()) {
                            Thread.sleep(200);
                        }
                        int count = reader.read(buf);
                        syncPrintln("Recv: " + (count > 0 ? new String(buf, 0, count) : "(none)"));
                    }
                } catch (IOException e) {
                    e.printStackTrace();
                } catch (InterruptedException e) {
                    syncPrintln("Receiver thread end");
                }
            });
            BufferedReader stdin = new BufferedReader(new InputStreamReader(System.in));
            PrintWriter writer = new PrintWriter(sock.getOutputStream());
            sendHeader(sock);
            receiver.start();
            System.out.println("Session start. Write text to send or /q to quit");
            while (true) {
                String msg = stdin.readLine();
                if (msg == null || "/q".equals(msg.trim())) {
                    break;
                }
                writer.println(msg);
                writer.flush();
                syncPrintln("Sent: " + msg);
            }
            receiver.interrupt();
            System.out.println("Session end");
        }
    }
    
    private final String host;
    private final int port;
    private final ProtocolHeader header;
    private final Object muxPrint;

    private SessionClient(String host, int port, ProtocolHeader header) {
        this.host = host;
        this.port = port;
        this.header = header;
        this.muxPrint = new Object();
    }

    public void exec() throws IOException {
        SocketClient.of(host, port, new SessionProc()).conn();
    }
    
    private void sendHeader(Socket sock) throws IOException {
        System.out.println("Header: " + header.toString());
        OutputStream out = sock.getOutputStream();
        byte[] b = header.getBytes();
        out.write(b);
        out.flush();
    }

    private void syncPrintln(String s) {
        synchronized (muxPrint) {
            System.out.println(s);
        }
    }
}
