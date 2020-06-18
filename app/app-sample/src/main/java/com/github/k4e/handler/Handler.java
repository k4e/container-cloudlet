package com.github.k4e.handler;

import java.io.IOException;
import java.net.Socket;

public abstract class Handler implements Runnable {

    private final Socket sock;

    public Handler(Socket sock) {
        this.sock = sock;
    }

    public Socket getSocket() {
        if (sock != null) {
            return sock;
        } else {
            throw new IllegalStateException("socket == null");
        }
    }

    protected void closeSocket() {
        try {
            if (sock != null) {
                sock.close();
            }
        } catch (IOException e) {
            e.printStackTrace();
        }
    }
}
