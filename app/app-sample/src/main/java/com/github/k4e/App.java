package com.github.k4e;

import com.github.k4e.handler.EchoHandlerFactory;
import com.google.common.base.Strings;

public class App {
    public static void main( String[] args ) {
        System.out.println("Build 2020-09-11");
        String envSleepMs = System.getenv("SLEEP_MS");
        int sleepMs = 0;
        if (!Strings.isNullOrEmpty(envSleepMs)) {
            sleepMs = Integer.parseInt(envSleepMs);
        }
        System.out.println("Started Echo Server App");
        new Server(8888).start(new EchoHandlerFactory(sleepMs));
    }
}
