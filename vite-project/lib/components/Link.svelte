<style>
    a {
        width: 100%;
        cursor: default;
        border: 0;
        text-decoration: none;
        background: transparent;
    }

    a:hover {
        cursor: default;
        text-decoration: none;
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

    /**
     * @typedef Props
     * @property {string} pageId
     * @property {import("svelte").Snippet} children
     * @property {"start"|"center"|"end"} [align]
     * @property {Record<string,string>} [fields]
     */

    /** @type {Props} */
    const {
        pageId,
        children,
        align = "start",
        fields = {},
        ...rest
    } = $props()
    const page = getContext("page")
    const path = getContext("path")

    /**
     * @param {Event} e
     */
    function onmouseup(e) {
        e.preventDefault()
        page(pageId, fields)
    }
</script>

<a href="{path(pageId, fields)}"
   class:start={"start"===align}
   class:center={"center"===align}
   class:end={"end"===align}
   {onmouseup}
   {...rest}
>
    {@render children()}
</a>