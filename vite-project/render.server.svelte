<script>
    //:app-imports
    import {setContext} from 'svelte'

    // Do not remove or discard `pageId`, it's being used by app-router.
    let {pageId, paths, path, data} = $props()
    setContext("Data", data)
    setContext("Page", function () {
        // Noop.
    })

    /**
     * @param {string} string
     */
    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    }

    setContext("Path", function (pageId) {
        let pathLocal = paths[pageId] ?? ''
        if (!paths[pageId]) {
            return ""
        }
        const resolved = {}
        for (let key in path) {
            resolved[key] = false
        }
        for (let key in path) {
            const value = data[key]
            const regex = escapeRegExp(`{${key}}`)
            let oldPath = pathLocal
            pathLocal = pathLocal.replaceAll(new RegExp(regex, 'g'), value)
            resolved[key] = oldPath === pathLocal
        }
        return pathLocal
    })

</script>

<!--app-router-->