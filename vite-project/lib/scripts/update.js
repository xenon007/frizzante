/**
 * @param {function(string):string} page
 * @param {function(string):string} path
 * @param {function(string,Record<string,any>)} navigate
 * @param {string} query
 * @param {Record<string,any>} state
 */
function done(
    page,
    path,
    navigate,
    query,
    state,
) {
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
                history.replaceState(
                    window.history.state ?? {},
                    "",
                    `${window.location.pathname}${document.location.hash}${query}`,
                )

                for (const key in state) {
                    delete state[key]
                }

                for (const key in data) {
                    state[key] = data[key]
                }

                if (response.redirected) {
                    navigate(page(response.url.replace(window.location.origin, "")), data)
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
 * @param {function(string):string} page
 * @param {function(string):string} path
 * @param {function(string,Record<string,any>)} navigate
 * @param {Record<string,any>} state
 * @return {function(e:SubmitEvent)}
 */
export function update(
    page,
    path,
    navigate,
    state,
) {
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
            fetch(query, {method, headers}).then(done(page, path, navigate, query, state)).catch(fail)
            return
        }

        fetch(
            formElement.action,
            {method, headers, body: formData}
        ).then(done(page, path, navigate, "", state)).catch(fail)
    }
}