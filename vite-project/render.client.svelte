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
         * @param {false|Record<string,any>} [data]
         */
        function (page, parameters, data = false) {
            navigate(page, "push", parameters, data)
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
     * @returns {{page:string,parameters:Record<string,string>}}
     */
    function _page(path) {
        const partsGiven = path.split("/")
        for (const page in pages) {
            const pathExpected = pages[page]
            const partsExpected = pathExpected.split("/")
            if (partsExpected.length !== partsGiven.length) {
                continue
            }

            /** @type {Record<string,string>} */
            const parameters = {}

            let ok = true
            for (let index = 0; index < partsExpected.length; index++) {
                const expectedIsParameter = partsExpected[index].startsWith("{") && partsExpected[index].endsWith("}")
                const givenAndExpectedAreDifferent = partsGiven[index] !== partsExpected[index]

                if (givenAndExpectedAreDifferent) {
                    if(!expectedIsParameter){
                        ok = false
                        break
                    }
                    const key = partsExpected[index].substring(0,partsExpected[index].length-1).substring(1)
                    parameters[key] = partsGiven[index]
                } else if(expectedIsParameter) {
                    // Given part and expected part cannot be equal while expected part is a parameter.
                    // We reject that.
                    ok = false
                    break
                }
            }

            if (ok) {
                return {
                    page,
                    parameters,
                }
            }
        }

        return {
            page: "",
            parameters: {}
        }
    }

    /**
     *
     * @param {string} page
     * @param {"back"|"forward"|"push"} modifier
     * @param {Record<string,string>} [parameters]
     * @param {false|Record<string,any>} [data]
     */
    function navigate(page, modifier, parameters, data = false) {
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

        if(false !== data){
            return
        }

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