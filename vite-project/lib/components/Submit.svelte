<style>
    .btn {
        width: 100%;
        border: 0;
        background: transparent;
    }

    .start {
        text-align: start;
    }

    .center {
        text-align: center;
    }

    .end {
        text-align: end;
    }
</style>

<script>
    import {update} from "../scripts/update.js";
    import {getContext} from "svelte";

    /**
     * @typedef Props
     * @property {import("svelte").Snippet} children
     * @property {"get" | "post" | "GET" | "POST"} [method]
     * @property {Record<string,string|number|boolean>} [form]
     * @property {"start"|"center"|"end"} [align]
     */

    /** @type {Props} */
    const {
        method = "GET",
        align = "start",
        form = {},
        children,
    } = $props()

</script>
<form {method} action="?" onsubmit={update(getContext("data"))}>
    {#each Object.keys(form) as key}
        {@const value = form[key]}
        <input type="hidden" name="{key}" value="{value}">
    {/each}
    <button
            type="submit"
            class="btn"
            class:start={"start"===align}
            class:center={"center"===align}
            class:end={"end"===align}
    >
        {@render children()}
    </button>
</form>