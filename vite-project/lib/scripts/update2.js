/**
 * @typedef Navigate
 * @property {string} page
 * @property {Record<string,string>} parameters
 * @property {string} location
 */

/**
 * @typedef ServerToClientSync
 * @property {Record<string,any>} data
 * @property {Navigate} [navigate]
 */

/**
 * @typedef DonePayload
 * @property {function(string,Record<string,string>)} navigate
 * @property {string} query
 * @property {Record<string,any>} data
 */

/**
 *
 * @param {DonePayload} payload
 */
function done(payload) {
    const {
        navigate,
        query,
        data,
    } = payload

    /**
     * @param {Response} response
     */
    return function (response) {
        if (response.status >= 300) {
            console.error(`Submit request failed with status ${response.status} ${response.statusText}.`)
            return
        }

        response.json()
            .then(
                /**
                 * @param {ServerToClientSync} sync
                 */
                function (sync) {
                    const responseData = sync.data ?? {}
                    const responseNavigate = sync.navigate ?? false
                    history.replaceState(window.history.state ?? {}, "", `${window.location.pathname}${document.location.hash}${query}`,)

                    for (const key in data) {
                        delete data[key]
                    }

                    for (const key in responseData) {
                        data[key] = responseData[key]
                    }

                    if (responseNavigate) {
                        navigate(responseNavigate.page, responseNavigate.parameters)
                    }
                }
            )
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
 * @typedef UpdatePayload
 * @property {function(string,Record<string,string>)} navigate
 * @property {Record<string,any>} data
 */

/**
 * @param {UpdatePayload} payload
 */
export function update(payload) {
    const { navigate, data} = payload
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
            const query = `?${search}`
            const init = {method, headers}
            const donePayload = {
                navigate,
                query,
                data,
            }
            fetch(query, init).then(done(donePayload)).catch(fail)
            return
        }

        const init = {method, headers, body: formData}
        const donePayload = {
            navigate,
            query: "",
            data,
        }

        fetch(formElement.action, init).then(done(donePayload)).catch(fail)
    }
}