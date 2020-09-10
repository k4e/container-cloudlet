package com.github.k4e;

import java.io.IOException;

import com.github.k4e.types.Request;
import com.google.common.collect.ImmutableMap;
import com.google.gson.Gson;

public class CloudletClient {
    
    public static final Request SAMPLE_CREATE = Request.create(
        new Request.Create("app-sample", "k4edev/app-sample:latest", 8888, 30088,
        ImmutableMap.of("SLEEP_MS", "5000"))
    );
    public static final Request SAMPLE_DELETE = Request.delete(
        new Request.Delete("app-sample")
    );

    public static void create(String host, int port) throws IOException {
        send(host, port, SAMPLE_CREATE);
    }

    public static void delete(String host, int port) throws IOException {
        send(host, port, SAMPLE_DELETE);
    }

    private static void send(String host, int port, Request req) throws IOException {
        Gson gson = new Gson();
        String msg = gson.toJson(req);
        send(host, port, msg);
    }

    private static void send(String host, int port, String msg) throws IOException {
        SocketSendRecv.sendRecv(host, port, msg);
    }
}
