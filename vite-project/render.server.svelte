<script>
    //:app-imports
    import {setContext} from 'svelte'

    /**
     * @typedef Props
     * @property {string} page
     * @property {Record<string,any>} data
     * @property {Record<string,string>} pages
     * @property {Record<string,string>} parameters
     */

    // Do not remove or discard `pageId`, it's being used by app-router.
    /** @type {Props} */
    let {page, data, pages, parameters} = $props()
    setContext("data", data)
    setContext("navigate", function () {
        // Noop.
    })

    /**
     * @param {string} string
     */
    function escapeRegExp(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    }

    setContext("path", path)
    setContext("page", _page)

    /**
     * @param {string} page
     * @param {Record<string,string>} [fields]
     */
    function path(page, fields = {}) {
        let result = pages[page] ?? ""
        if (!pages[page]) {
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
</script>

<!--app-router-->