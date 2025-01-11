import { hydrate } from "svelte";
import RenderClient from "./render.client.svelte";
// @ts-ignore
hydrate(RenderClient, { target: target(), props: props() });
