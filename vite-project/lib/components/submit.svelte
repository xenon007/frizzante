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

    const path = getContext("path")
    const onsubmit = update(getContext("data"))
    const id = uuid()

    /**
     * @typedef Props
     * @property {string} [action]
     * @property {import("svelte").Snippet} children
     * @property {Record<string,string|number|boolean>} [form]
     */

    /** @type {Props} */
    let {
        action = '?',
        children,
        form = {},
    } = $props()


    if('?' !== action){
        action = path(action)
    }
</script>

<form method="POST" {action} {onsubmit}>
    {#each Object.keys(form) as key}
        {@const value = form[key]}
        <input type="hidden" name="{key}" value="{value}">
    {/each}

    <input class="submit" type="submit" id="{id}"/>

    <label for="{id}">
        {@render children()}
    </label>
</form>
