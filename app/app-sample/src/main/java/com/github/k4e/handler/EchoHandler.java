package com.github.k4e.handler;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;
import java.util.Arrays;

public class EchoHandler extends Handler {
    
    private static final int BUF_SIZE = 1024 * 1024;

    private final int sleepMs;

    public EchoHandler(Socket sock, int sleepMs) {
        super(sock);
        this.sleepMs = sleepMs;
    }

    @Override
    public void run() {
        char[] buf = new char[BUF_SIZE];
        try {
            System.out.printf("[Accepted %s]\n", getSocket().getInetAddress());
            InputStreamReader reader = new InputStreamReader(getSocket().getInputStream());
            PrintWriter writer = new PrintWriter(getSocket().getOutputStream());
            while (!getSocket().isClosed()) {
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
            System.out.printf("[Closing %s]\n", getSocket().getInetAddress());
            closeSocket();
        }
    }
}
