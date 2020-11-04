package com.github.k4e;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;
import java.net.UnknownHostException;

public class SessionClient {

    public static SessionClient of(String host, int port)
    throws UnknownHostException {
        return new SessionClient(host, port);
    }

    private final String host;
    private final int port;
    private final Object muxPrint;

    private SessionClient(String host, int port) {
        this.host = host;
        this.port = port;
        this.muxPrint = new Object();
    }

    public void exec() throws IOException {
        Socket sock = null;
        try {
            sock = new Socket(host, port);
            System.out.println("Connection open");
            accept(sock);
        } finally {
            if (sock != null) {
                sock.close();
                System.out.println("Connection closed");
            }
        }
    }
    
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
                    if (count < 0) {
                        System.out.println("Read reached end");
                        break;
                    }
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
        receiver.start();
        System.out.println("Session start. Write text to send or /q to quit");
        while (receiver.isAlive()) {
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

    private void syncPrintln(String s) {
        synchronized (muxPrint) {
            System.out.println(s);
        }
    }
}
