<script>
    //:app-imports
    import {setContext} from "svelte";

    let {pageId, paths, path, data} = $props()
    // Do not remove or discard `pageId`, it's being used by app-router.
    let pageIdState = $state(pageId)
    let dataState = $state({...data})
    let navCounterPrevious = 0
    setContext("data", dataState)
    setContext("page",
        /**
         * @param {string} pageIdLocal
         * @param {Record<string,string>} [fields]
         */
        function (pageIdLocal, fields) {
            pageFn(pageIdLocal, "push", fields)
        }
    )
    setContext("path", pathFn)

    window.history.replaceState({
        ...(window.history.state ?? {}),
        pageId,
        fields: path,
        navCounter: navCounterPrevious,
    }, "", `${document.location.pathname}${document.location.hash}${document.location.search}`)

    window.addEventListener("popstate", (e) => {
        e.preventDefault()
        const pageIdLocal = e.state?.pageId ?? ""
        const fields = e.state?.fields ?? {}
        const navCounterLocal = e.state?.navCounter ?? 0
        if (navCounterLocal < navCounterPrevious) {
            pageFn(pageIdLocal, "back", fields)
            navCounterPrevious = navCounterLocal
        } else if (navCounterLocal > navCounterPrevious) {
            pageFn(pageIdLocal, "forward", fields)
            navCounterPrevious = navCounterLocal
        } else {
            pageFn(pageIdLocal, "push", fields)
        }
    });

    /**
     * @param {string} string
     */
    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    }

    /**
     * @param {string} pageId
     * @param {Record<string,string>} [fields]
     */
    function pathFn(pageId, fields = {}) {
        let result = paths[pageId] ?? ""
        if (!paths[pageId]) {
            return ""
        }

        for (let key in fields) {
            const value = fields[key]
            const regex = escapeRegExp(`{${key}}`)
            result = result.replaceAll(new RegExp(regex, "g"), value)
        }

        return result
    }

    /**
     *
     * @param {string} pageIdLocal
     * @param {"back"|"forward"|"push"} modifier
     * @param {Record<string,string>} fields
     */
    function pageFn(pageIdLocal, modifier, fields) {
        if (!paths[pageIdLocal]) {
            return
        }

        const pathLocal = pathFn(pageIdLocal, fields)
        if ("push" === modifier) {
            window.history.pushState({
                pageId: pageIdLocal,
                fields: fields,
                navCounter: ++navCounterPrevious,
            }, "", pathLocal);
        }
        pageIdState = pageIdLocal

        fetch(pathLocal, {headers: {"Accept": "application/json"}}).then(async (response) => {
            const data = await response.json()

            for (const key in dataState) {
                delete dataState[key]
            }

            for (const key in data) {
                dataState[key] = data[key]
            }
        })
    }
</script>

<!--app-router-->