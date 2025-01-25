import { mount } from "svelte";
import RenderClient from "./render.client.svelte";
// @ts-ignore
target().innerHTML = "";
mount(RenderClient, { target: target(), props: props() });
