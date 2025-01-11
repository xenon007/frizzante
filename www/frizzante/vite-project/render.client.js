import { hydrate } from "svelte";
import RenderServer from "./render.server.svelte";
// @ts-ignore
hydrate(RenderServer, { target: target(), props: props() });
