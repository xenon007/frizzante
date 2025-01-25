import { mount } from "svelte";
import RenderClient from "./render.client.svelte";
// @ts-ignore
mount(RenderClient, { target: target(), props: props() });
