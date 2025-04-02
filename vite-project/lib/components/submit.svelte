<style>
    form {
        position: relative;
        width: 100%;
        height: 100%;
    }
    .submit {
        display: none;
    }
</style>

<script>
    import {update} from "../scripts/update.js";
    import {getContext} from "svelte";
    import {uuid} from "../scripts/uuid.js";

    /**
     * @typedef Props
     * @property {import("svelte").Snippet} children
     * @property {"get" | "post" | "GET" | "POST"} [method]
     * @property {Record<string,string|number|boolean>} [form]
     */

    /** @type {Props} */
    const {
        method = "GET",
        form = {},
        children,
    } = $props()

    const id = uuid()
</script>

<form {method} action="?" onsubmit={update(getContext("data"))}>
    {#each Object.keys(form) as key}
        {@const value = form[key]}
        <input type="hidden" name="{key}" value="{value}">
    {/each}

    <input class="submit" type="submit" id="{id}"/>

    <label for="{id}">
        {@render children()}
    </label>
</form>