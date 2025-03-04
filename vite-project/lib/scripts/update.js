/**
 * @typedef Props
 * @property {import("svelte").Snippet} children
 * @property {"get" | "post" | "GET" | "POST"} [method]
 * @property {Record<string,string|number|boolean>} [form]
 * @property {"start"|"center"|"end"} [align]
 */

/**
 * @param {string} queryString
 * @param {Record<string,any>} dataState
 */
function done(queryString, dataState) {
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
 *
 * @param {Record<string,any>} state
 * @return {function(e:SubmitEvent)}
 */
export function update(state) {
    return function onsubmit(e) {
        e.preventDefault()
        /** @type {HTMLFormElement} */
        const formElement = e.target
        const formData = new FormData(formElement)
        const method = formElement.method
        const headers = {"Accept": "application/json"}

        if (method === "get" || method === "GET") {
            const data = new URLSearchParams();
            for (const [key, value] of formData) {
                data.append(key, value.toString());
            }

            const search = data.toString()
            const queryString = `?${search}`
            fetch(queryString, {method, headers}).then(done(queryString, state)).catch(fail)
            return
        }

        fetch("?", {method, headers, body: formData}).then(done("", state))
    }
}