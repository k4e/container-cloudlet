package com.github.k4e.types;

import java.io.Serializable;
import java.util.Map;

public class Request implements Serializable {

    public static Request create(Create create) {
        return new Request("create", create, null);
    }

    public static Request delete(Delete delete) {
        return new Request("delete", null, delete);
    }

    public static class Create implements Serializable {
        public String name;
        public String image;
        public Integer port;
        public Integer extPort;
        public Map<String, String> env;
        public Create(String name, String image, int port, int extPort, Map<String, String> env) {
            this.name = name;
            this.image = image;
            this.port = port;
            this.extPort = extPort;
            this.env = env;
        }
    }

    public static class Delete implements Serializable {
        public String name;
        public Delete(String name) {
            this.name = name;
        }
    }

    public String op;
    public Create create;
    public Delete delete;

    private Request(String op, Create create, Delete delete) {
        this.op = op;
        this.create = create;
        this.delete = delete;
    }
}
