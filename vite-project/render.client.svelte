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
         */
        function (pageIdLocal) {
            pageFn(pageIdLocal, "push")
        }
    )
    setContext("path", pathFn)

    window.history.replaceState({
        ...(window.history.state ?? {}),
        pageId, navCounter: navCounterPrevious
    }, "", document.location.pathname)

    window.addEventListener("popstate", (e) => {
        e.preventDefault()
        const pageIdLocal = e.state?.pageId ?? ""
        const navCounterLocal = e.state?.navCounter ?? 0
        if (navCounterLocal < navCounterPrevious) {
            pageFn(pageIdLocal, "back")
            navCounterPrevious = navCounterLocal
        } else if (navCounterLocal > navCounterPrevious) {
            pageFn(pageIdLocal, "forward")
            navCounterPrevious = navCounterLocal
        } else {
            pageFn(pageIdLocal, "push")
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
     */
    function pathFn(pageId) {
        let result = paths[pageId] ?? ""
        if (!paths[pageId]) {
            return ""
        }
        const resolved = {}
        for (let key in path) {
            resolved[key] = false
        }
        for (let key in path) {
            const value = dataState[key]
            const regex = escapeRegExp(`{${key}}`)
            let oldPath = result
            result = result.replaceAll(new RegExp(regex, "g"), value)
            resolved[key] = oldPath === result
        }

        return result
    }

    /**
     *
     * @param {string} pageIdLocal
     * @param {"back"|"forward"|"push"} modifier
     */
    function pageFn(pageIdLocal, modifier) {
        if (!paths[pageIdLocal]) {
            return
        }

        const pathLocal = pathFn(pageIdLocal)
        if ("push" === modifier) {
            window.history.pushState({pageId: pageIdLocal, navCounter: ++navCounterPrevious}, "", pathLocal);
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