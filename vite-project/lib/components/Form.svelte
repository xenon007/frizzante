<script>
    import {update} from "../scripts/update.js";
    import {getContext} from "svelte";

    /**
     * @typedef Props
     * @property {import("svelte").Snippet} children
     * @property {"get" | "post" | "GET" | "POST"} [method]
     * @property {string} [action]
     */

    /** @type {Props} */
    const {method = "GET", action = '?', ...rest} = $props()
</script>

<form {method} {action} onsubmit={update(getContext("data"))} {...rest}>
    {@render children()}
</form>