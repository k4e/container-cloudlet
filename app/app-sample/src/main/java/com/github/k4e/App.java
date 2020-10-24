package com.github.k4e;

import com.google.common.base.Strings;

public class App {
    public static void main( String[] args ) {
        System.out.println("Build 2020-10-24.1");
        String envSleepMs = System.getenv("SLEEP_MS");
        int sleepMs = 0;
        if (!Strings.isNullOrEmpty(envSleepMs)) {
            sleepMs = Integer.parseInt(envSleepMs);
        }
        System.out.println("Started echo server");
        new EchoServer(8888, sleepMs).start();
    }
}
