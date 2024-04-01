import { task } from "@maf/core.ts";
import { containerStop, run } from "@maf/docker/mod.ts";
import { getGoBuildEnv, GOARCH, GoBuild, GOOS } from "@maf/lang/go.ts";
import {
  CommitStatus,
  getClient,
  Release,
  webhook,
} from "@maf/service/github.ts";
import { format as formatBytes } from "@std/fmt/bytes.ts";

const goVersion = "1.22";
const matrix = [
  { os: GOOS.linux, arch: GOARCH.amd64 },
  { os: GOOS.linux, arch: GOARCH.arm64 },
  { os: GOOS.windows, arch: GOARCH.amd64 },
];

const testConfig = `
concurrency = 8

output "web" {
  path           = "/out"
  include_static = true
}

layer "normal" {
  render = "pixel"
}

layer "biome" {
  render  = "biome"
  opacity = 0.5
}

layer "light" {
  render = "light"
}

map "test" {
  output = "web"
  path   = "/repository/data/world/region"
  layers = ["normal", "biome", "light"]
}
`;

export const test = task("test", async () => {
  await run(
    null,
    {
      image: `itzg/minecraft-server`,
      env: ["EULA=TRUE"],
      createOpts: {
        hostConfig: {
          binds: [`${Deno.cwd()}/data:/data`],
        },
      },
      onStreamEvent: async (id, e) => {
        if (
          e.type === "stdout" && e.data.includes("[Server thread/INFO]: Done")
        ) {
          await containerStop(id);
        }
      },
    },
  );

  await run(
    `go run cmd/carto/main.go build --config /config.hcl`,
    {
      image: `golang:${goVersion}-alpine`,
      env: getGoBuildEnv({ os: GOOS.linux, arch: GOARCH.amd64 }),
      files: {
        "/config.hcl": testConfig,
      },
    },
  );
});

export async function buildAll() {
  await Promise.all(matrix.map((it) => build.call({ opts: { go: it } })));
}

const build = task("build", async ({ opts, release, sha }: {
  opts?: { go: GoBuild; version?: string };
  release?: Release;
  sha?: string;
}) => {
  const client = await getClient();
  const go = opts?.go || { os: GOOS.linux, arch: GOARCH.amd64 };

  let name = `carto-${go.os}-${go.arch}`;
  if (go.os === "windows") {
    name = name + ".exe";
  }

  let commitStatus = null;
  if (client && sha) {
    commitStatus = await CommitStatus.create("b1naryth1ef/carto", sha, {
      state: "pending",
      context: `carto-${go.os}-${go.arch}`,
    });
  }

  const res = await run(
    `go build -o ${name} cmd/carto/main.go`,
    {
      image: `golang:${opts?.version || goVersion}-alpine`,
      env: getGoBuildEnv(go),
    },
  );

  const { size } = await Deno.stat(name);
  await commitStatus?.update({
    state: "success",
    description: `${formatBytes(size)} took ${
      (res.timings.run / 1000).toFixed(2)
    }s`,
  });

  if (release) {
    if (client === null) {
      throw new Error(`failed to get github client`);
    }

    await client.uploadReleaseAsset(
      release,
      name,
      await Deno.readFile(name),
    );
  }
});

export const github = webhook(async (event) => {
  if (event.push && event.push.head_commit) {
    test.spawn(undefined, { ref: event.push.head_commit.id });

    for (const variant of matrix) {
      build.spawn({
        opts: { go: variant },
        sha: event.push.head_commit.id,
      }, { ref: event.push.head_commit.id });
    }
  } else if (event.create) {
    const client = await getClient();
    if (client === null) {
      throw new Error(`failed to get github client`);
    }

    if (event.create.ref_type === "tag" && event.create.ref.startsWith("v")) {
      const release = await client.createRelease("b1naryth1ef/carto", {
        tag: event.create.ref,
        name: event.create.ref,
        draft: true,
      });

      for (const variant of matrix) {
        build.spawn({
          opts: { go: variant },
          release: release,
        }, { ref: event.create.ref });
      }
    }
  }
});
