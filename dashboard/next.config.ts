import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  output: "standalone",
  outputFileTracingRoot: path.join(process.cwd(), ".."),
  experimental: {
    optimizePackageImports: ["@xyflow/react", "lucide-react"],
  },
};

export default nextConfig;
