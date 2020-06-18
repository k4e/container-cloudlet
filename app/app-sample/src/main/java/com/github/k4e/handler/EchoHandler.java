package com.github.k4e.handler;

import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.net.Socket;

public class EchoHandler extends Handler {
    
    public EchoHandler(Socket sock) {
        super(sock);
    }

    @Override
    public void run() {
        try {
            System.out.printf("[Accepted %s]\n", getSocket().getInetAddress());
            InputStreamReader reader = new InputStreamReader(getSocket().getInputStream());
            PrintWriter writer = new PrintWriter(getSocket().getOutputStream());
            char[] buf = new char[1024];
            while (!getSocket().isClosed()) {
                int count = reader.read(buf);
                if (count < 1) {
                    break;
                }
                System.out.print(new String(buf, 0, count));
                writer.print(buf);
                writer.flush();
            }
        } catch (IOException e) {
            e.printStackTrace();
        } finally {
            System.out.printf("[Closing %s]\n", getSocket().getInetAddress());
            closeSocket();
        }
    }
}
