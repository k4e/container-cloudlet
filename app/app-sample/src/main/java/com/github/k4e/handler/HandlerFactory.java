package com.github.k4e.handler;

import java.net.Socket;

public abstract class HandlerFactory {
    
    public abstract Handler create(Socket sock);
}
