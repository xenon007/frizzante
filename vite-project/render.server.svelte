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
</script>

<!--app-router-->