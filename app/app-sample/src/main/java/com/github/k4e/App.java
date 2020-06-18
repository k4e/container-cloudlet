package com.github.k4e;

import com.github.k4e.handler.EchoHandlerFactory;

public class App {
    public static void main( String[] args ) {
        System.out.println("Started Echo Server App");
        new Server(8888).start(new EchoHandlerFactory());
    }
}
