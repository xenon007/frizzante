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
    import {getContext} from "svelte";

    const dataState = getContext("data")

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
        children
    } = $props()

    /** @type {HTMLFormElement} */
    let formElement

    /**
     * @param {string} queryString
     */
    function done(queryString) {
        /**
         * @param {Response} response
         */
        return function (response) {
            if (response.status >= 300) {
                console.error(`Submit request failed with status ${response.status} ${response.statusText}.`)
                return
            }

            response.json()
                .then(function (data) {
                    const state = window.history.state ?? {}
                    history.replaceState(state, "", `${window.location.pathname}${document.location.hash}${queryString}`)
                    for (const key in dataState) {
                        delete dataState[key]
                    }

                    for (const key in data) {
                        dataState[key] = data[key]
                    }

                })
                .catch(fail)
        }
    }

    /**
     * @param {any} reason
     */
    function fail(reason) {
        console.error("Submit request failed.", reason)
    }

    /**
     * @param {Event} e
     */
    function onsubmit(e) {
        e.preventDefault()
        const headers = {"Accept": "application/json"}
        const form = new FormData(formElement)

        if (method === "get" || method === "GET") {
            const data = new URLSearchParams();
            for (const [key, value] of form) {
                data.append(key, value.toString());
            }
            const search = data.toString()
            const queryString = `?${search}`
            fetch(queryString, {method, headers}).then(done(queryString)).catch(fail)
            return
        }

        fetch("?", {method, headers, body: form}).then(done(""))
    }
</script>
<form bind:this={formElement} {method} action="?" {onsubmit}>
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