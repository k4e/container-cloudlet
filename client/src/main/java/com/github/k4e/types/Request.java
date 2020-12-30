package com.github.k4e.types;

import java.io.Serializable;
import java.util.Map;

public class Request implements Serializable {
    private static final long serialVersionUID = -8180504938175762670L;

    public static Request deploy(Deploy deploy) {
        return new Request("deploy", deploy, null);
    }

    public static Request remove(Remove remove) {
        return new Request("remove", null, remove);
    }

    public static class Deploy implements Serializable {
        private static final long serialVersionUID = -204586336929613360L;
        public enum Type {
            NEW("new"), FWD("fwd"), LM("lm"), FWDLM("fwdlm");
            public static Type valueOfIgnoreCase(String s) {
                for (Type t : values()) {
                    if (t.name().equalsIgnoreCase(s)) {
                        return t;
                    }
                }
                return null;
            }
            private final String s;
            private Type(String s) {
                this.s = s;
            }
            public String getTypeName() {
                return s;
            }
        }
        public static class Port implements Serializable {
            private static final long serialVersionUID = 4028722031667672178L;
            public Integer in;
            public Integer ext;
            public Port(int in, int ext) {
                this.in = in;
                this.ext = ext;
            }
        }
        public static class NewApp implements Serializable {
            private static final long serialVersionUID = -7084246250409417126L;
            public String image;
            public Port port;
            public Map<String, String> env;
            public NewApp(String image, Port port, Map<String, String> env) {
                this.image = image;
                this.port = port;
                this.env = env;
            }
        }
        public static class Fwd implements Serializable {
            private static final long serialVersionUID = 661156972888154013L;
            public String srcAddr;
            public Port port;
            public Fwd(String srcAddr, Port port) {
                this.srcAddr = srcAddr;
                this.port = port;
            }
        }
        public static class LM implements Serializable {
            private static final long serialVersionUID = -4857209607926377458L;
            public String image;
            public String srcAddr;
            public String srcName;
            public Port port;
            public String dstAddr;
            public Integer bwLimit;
            public LM(String image, String srcAddr, String srcName, Port port, String dstAddr, int bwLimit) {
                this.image = image;
                this.srcAddr = srcAddr;
                this.srcName = srcName;
                this.port = port;
                this.dstAddr = dstAddr;
                this.bwLimit = bwLimit;
            }
        }
        public static class FwdLM implements Serializable {
            private static final long serialVersionUID = -678769254653996184L;
            public String image;
            public String srcAddr;
            public String srcName;
            public Integer srcPort;
            public Port port;
            public String dstAddr;
            public Integer bwLimit;
            public FwdLM(String image, String srcAddr, String srcName, int srcPort, Port port, String dstAddr, int bwLimit) {
                this.image = image;
                this.srcAddr = srcAddr;
                this.srcName = srcName;
                this.srcPort = srcPort;
                this.port = port;
                this.dstAddr = dstAddr;
                this.bwLimit = bwLimit;
            }
        }
        public String name;
        public String type;
        public NewApp newApp;
        public Fwd fwd;
        public LM lm;
        public FwdLM fwdlm;
        public Deploy(String name, Type type, NewApp newApp, Fwd fwd, LM lm, FwdLM fwdlm) {
            this.name = name;
            this.type = type.getTypeName();
            this.newApp = newApp;
            this.fwd = fwd;
            this.lm = lm;
            this.fwdlm = fwdlm;
        }
    }

    public static class Remove implements Serializable {
        private static final long serialVersionUID = -7285750816295865630L;
        public String name;
        public Remove(String name) {
            this.name = name;
        }
    }

    public String method;
    public Deploy deploy;
    public Remove remove;

    private Request(String method, Deploy deploy, Remove remove) {
        this.method = method;
        this.deploy = deploy;
        this.remove = remove;
    }
}
