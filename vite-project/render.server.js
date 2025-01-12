import { render as _render } from "svelte/server";
import RenderServer from "./render.server.svelte";
export async function render(props) {
  return _render(RenderServer, { props });
}
