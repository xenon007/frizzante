import { hydrate } from "svelte";
import RenderClient from "./render.client.svelte";
// @ts-ignore
target().innerHTML = "";
hydrate(RenderClient, { target: target(), props: props() });
