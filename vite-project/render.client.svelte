<script>
    //:app-imports
    import {setContext} from "svelte";

    /**
     * @typedef Props
     * @property {string} page
     * @property {Record<string,any>} data
     * @property {Record<string,string>} pages
     * @property {Record<string,string>} parameters
     */

    /** @type {Props} */
    let {page, data, pages, parameters} = $props()
    // Do not remove or discard `pageId`, it's being used by app-router.
    let pageState = $state(page)
    let dataState = $state({...data})
    let navCounterPrevious = 0
    setContext("data", dataState)
    setContext("navigate",
        /**
         * @param {string} page
         * @param {Record<string,string>} [parameters]
         */
        function (page, parameters) {
            navigate(page, "push", parameters)
        }
    )
    setContext("path", path)
    setContext("page", _page)

    window.history.replaceState({
        ...(window.history.state ?? {}),
        page,
        parameters,
        navCounter: navCounterPrevious,
    }, "", `${document.location.pathname}${document.location.hash}${document.location.search}`)

    window.addEventListener("popstate", (e) => {
        e.preventDefault()
        const pageLocal = e.state?.page ?? ""
        const parameters = e.state?.parameters ?? {}
        const navCounterLocal = e.state?.navCounter ?? 0
        if (navCounterLocal < navCounterPrevious) {
            navigate(pageLocal, "back", parameters)
            navCounterPrevious = navCounterLocal
        } else if (navCounterLocal > navCounterPrevious) {
            navigate(pageLocal, "forward", parameters)
            navCounterPrevious = navCounterLocal
        } else {
            navigate(pageLocal, "push", parameters)
        }
    });

    /**
     * @param {string} string
     */
    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    }

    /**
     * @param {string} page
     * @param {Record<string,string>} [parameters]
     */
    function path(page, parameters = {}) {
        let result = pages[page] ?? ""
        if (!pages[page]) {
            return ""
        }

        for (let key in parameters) {
            const value = parameters[key]
            const regex = escapeRegExp(`{${key}}`)
            result = result.replaceAll(new RegExp(regex, "g"), value)
        }

        return result
    }

    /**
     * @param {string} path
     */
    function _page(path) {
        const parts = path.split("/")
        for (const page in pages) {
            const pagePathLocal = pages[page]
            if(pagePathLocal === path) {
                return page
            }

            const partsLocal = pagePathLocal.split("/")
            if(partsLocal.length !== parts.length){
                continue
            }

            let ok = true
            for (const index in partsLocal) {
                if(partsLocal[index] !== parts[index] && !parts[index].startsWith("{")){
                    ok = false
                    break
                }
            }

            if(ok){
                return page
            }
        }

        return ""
    }

    /**
     *
     * @param {string} page
     * @param {"back"|"forward"|"push"} modifier
     * @param {Record<string,string>} [parameters]
     */
    function navigate(page, modifier, parameters) {
        if (!pages[page]) {
            return
        }

        const pathLocal = path(page, parameters)
        if ("push" === modifier) {
            window.history.pushState({
                page,
                parameters,
                navCounter: ++navCounterPrevious,
            }, "", pathLocal);
        }
        pageState = page

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