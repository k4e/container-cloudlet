package com.github.k4e;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.ServerSocket;
import java.net.Socket;
import java.net.SocketException;
import java.net.InetAddress;
import java.net.NetworkInterface;
import java.util.Arrays;
import java.util.Collections;
import java.util.Enumeration;

public class EchoServer {
    private static final int BUF_SIZE = 32 * 1024 * 1024;

    private final int port;
    private final int sleepMs;

    public EchoServer(int port, int sleepMs) {
        this.port = port;
        this.sleepMs = sleepMs;
    }

    public void start() {
        try (ServerSocket sv = new ServerSocket(port)) {
            displayAddress();
            while (true) {
                try {
                    Socket sock = sv.accept();
                    Thread th = new Thread(new SocketHandler(sock));
                    th.start();
                } catch (IOException e) {
                    e.printStackTrace();
                }
            }
        } catch (IOException e) {
            e.printStackTrace();
        }
    }

    public void displayAddress() throws SocketException {
        Enumeration<NetworkInterface> ifaces = NetworkInterface.getNetworkInterfaces();
        if (ifaces != null) {
            for (NetworkInterface iface : Collections.list(ifaces)) {
                System.out.printf("%s\t", iface.getName());
                for (InetAddress addr : Collections.list(iface.getInetAddresses())) {
                    System.out.printf("%s ", addr);
                }
                System.out.println();
            }
        } else {
            System.out.println("No interfaces");
        }
    }

    class SocketHandler implements Runnable {
        private Socket sock;
        public SocketHandler(Socket sock) {
            this.sock = sock;
        }
        @Override public void run() {
            char[] buf = new char[BUF_SIZE];
            try {
                System.out.printf("[Accepted %s]\n", sock.getInetAddress());
                InputStreamReader reader = new InputStreamReader(sock.getInputStream());
                PrintWriter writer = new PrintWriter(sock.getOutputStream());
                while (!sock.isClosed()) {
                    int count = reader.read(buf);
                    if (count < 1) {
                        break;
                    }
                    if (sleepMs > 0) {
                        Thread.sleep(sleepMs);
                    }
                    char[] output = Arrays.copyOfRange(buf, 0, count);
                    writer.print(output);
                    writer.flush();
                }
            } catch (IOException e) {
                e.printStackTrace();
            } catch (InterruptedException e) {
                e.printStackTrace();
            } finally {
                System.out.printf("[Closing %s]\n", sock.getInetAddress());
                try {
                    sock.close();
                } catch (IOException e) {
                    e.printStackTrace();
                }
            }
        }
    }
}
