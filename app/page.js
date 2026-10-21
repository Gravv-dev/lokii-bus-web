import { readFileSync } from "node:fs";
import { join } from "node:path";
import Interactions from "./interactions";

const source = readFileSync(join(process.cwd(), "index.html"), "utf8");
const body = source.match(/<body[^>]*>([\s\S]*?)<\/body>/i);

if (!body) {
  throw new Error("Could not find the page body in index.html");
}

const markup = body[1].replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "");

export default function Home() {
  return (
    <>
      <div dangerouslySetInnerHTML={{ __html: markup }} />
      <Interactions />
    </>
  );
}
