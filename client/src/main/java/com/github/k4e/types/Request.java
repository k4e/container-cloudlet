package com.github.k4e.types;

import java.io.Serializable;
import java.util.Map;

public class Request implements Serializable {
    private static final long serialVersionUID = -8180504938175762670L;

    public static Request create(Create create) {
        return new Request("create", create, null);
    }

    public static Request delete(Delete delete) {
        return new Request("delete", null, delete);
    }

    public static class Create implements Serializable {
        private static final long serialVersionUID = -204586336929613360L;
        public String name;
        public Boolean createApp;
        public String image;
        public Integer port;
        public Integer extPort;
        public Map<String, String> env;
        public Create(String name, boolean createApp, String image, int port, int extPort,
                Map<String, String> env) {
            this.name = name;
            this.createApp = createApp;
            this.image = image;
            this.port = port;
            this.extPort = extPort;
            this.env = env;
        }
    }

    public static class Delete implements Serializable {
        private static final long serialVersionUID = -7285750816295865630L;
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
