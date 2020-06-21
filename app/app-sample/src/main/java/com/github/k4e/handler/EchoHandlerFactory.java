package com.github.k4e.handler;

import java.net.Socket;

public class EchoHandlerFactory extends HandlerFactory {
    
    private final int sleepMs;

    public EchoHandlerFactory(int sleepMs) {
        this.sleepMs = sleepMs;
    }

    @Override
    public EchoHandler create(Socket sock) {
        return new EchoHandler(sock, sleepMs);
    }
}