<script>
    import {getContext} from "svelte";
    import {update} from "../scripts/update.js";

    const path = getContext("path")
    const onsubmit = update(getContext("data"))

    /**
     * @typedef Props
     * @property {import("svelte").Snippet} children
     * @property {string} [action]
     */

    /** @type {Props} */
    let {children, action = '?', ...rest} = $props()

    if ('?' !== action) {
        action = path(action)
    }

</script>

<form method="POST" {action} {...rest} {onsubmit}>
    {@render children()}
</form>
