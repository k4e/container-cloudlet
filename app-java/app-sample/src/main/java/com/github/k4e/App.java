package com.github.k4e;

import java.util.Random;

import com.google.common.base.Strings;

public class App {

    private static final Random RANDOM = new Random();
    private static volatile byte[] padding;

    public static void main( String[] args ) {
        System.out.println("Build 2021-01-03");
        String envSleepMs = System.getenv("SLEEP_MS");
        int sleepMs = 0;
        if (!Strings.isNullOrEmpty(envSleepMs)) {
            sleepMs = Integer.parseInt(envSleepMs);
        }
        if (sleepMs > 0) {
            System.out.printf("Sleep: %d ms\n", sleepMs);
        }
        String envPadMB = System.getenv("PADDING");
        int padMB = 0;
        if (!Strings.isNullOrEmpty(envPadMB)) {
            padMB = Integer.parseInt(envPadMB);
        }
        if (padMB > 0) {
            System.out.printf("Padding memory size: %d MiB\n", padMB);
            padding = new byte[padMB * 1024 * 1024];
            for (int i = 0; i < padding.length; ++i) {
                padding[i] = (byte)RANDOM.nextInt(Byte.MAX_VALUE + 1);
            }
        }
        boolean upstreamMode = "UP".equalsIgnoreCase(System.getenv("DIRECTION"));
        if (upstreamMode) {
            System.out.println("Direction: up");
        }
        System.out.println("Started echo server");
        new EchoServer(8888, sleepMs, upstreamMode).start();
    }
}
