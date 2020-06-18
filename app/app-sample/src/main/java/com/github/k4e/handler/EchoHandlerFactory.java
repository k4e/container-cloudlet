package com.github.k4e.handler;

import java.net.Socket;

public class EchoHandlerFactory extends HandlerFactory {
    
    @Override
    public EchoHandler create(Socket sock) {
        return new EchoHandler(sock);
    }
}