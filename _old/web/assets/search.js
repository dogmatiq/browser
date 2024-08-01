const searchForm = document.getElementById('search')
const searchInput = searchForm.getElementsByTagName('input')[0]
const searchResults = document.getElementById('search_results')
const searchItems = (async () => {
    const res = await window.fetch('/search/items.json')
    return res.json()
})()


function filterItems(query, items) {
    const matches = []

    for (item of items) {
        const indices = findMatchingRegions(item.name, query)
        if (indices.length !== 0) {
            const score = computeScore(item.name, indices)
            matches.push({score, indices, item})
        }
    }

    matches.sort((a, b) => {
        if (a.score !== b.score) {
            return a.score - b.score
        }

        return a.item.name.localeCompare(b.item.name)
    })

    return matches
}

// computeScore returns a score for a item that matched a search query.
// The lower the score, the better the match.
function computeScore(name, indices) {
    const lengthWeight = 1.0 
    const distanceWeight = 1.0
    const disjointWeight = 1.0

    // Move shorter matches to the top, as this means the query string matched a
    // greater percentage of the name.
    let score = name.length * lengthWeight

    // Move results with matches closer to the start of the string to the top,
    // as you would _usually_ start by typing the start of the word.
    for (const [begin, end] of indices) {
        score += (begin / name.length) * distanceWeight
        break
    }

    // Add a further penalty for having more disjoint matches.
    score *= indices.length * disjointWeight

    return score
}

function findMatchingRegions(name, query) {
    name = name.toLowerCase()
    query = query.toLowerCase()

    const indices = []

    let nameIndex = 0
    let matchIndex = 0
    let withinMatch = false

    while (name && query) {
        const queryChar = query[0]
        if (queryChar.match(/\s/)) {
            query = query.substr(1)
            continue
        }

        if (!withinMatch) {
            const i = name.indexOf(query)
            if (i !== -1) {
                indices.push([nameIndex+i, nameIndex+i + query.length - 1])
                return indices
            }
        }

        const nameChar = name[0]
        nameIndex++
        name = name.substr(1)

        if (nameChar.match(/\s/)) {
            continue
        }

        if (nameChar === queryChar) {
            query = query.substr(1)    
            
            if (!withinMatch) {
                withinMatch = true
                matchIndex = nameIndex - 1
            }
        } else if (withinMatch) {
            withinMatch = false
            indices.push([matchIndex, nameIndex - 2])
        }
    }

    if (query) {
        return [];
    }

    if (withinMatch) {
        indices.push([matchIndex, nameIndex - 1])
    }

    return indices;
}

searchForm.addEventListener("submit", (event) => {
    event.preventDefault()

    const group = searchResults.getElementsByClassName("list-group")[0]
    let active = group.getElementsByClassName('active')[0]

    if (!active) {
        active = group.getElementsByClassName('result')[0]
    }

    if (active) {
        active.getElementsByTagName('a')[0].click()
    }
})

searchInput.addEventListener("search", doSearch)
searchInput.addEventListener("keyup", doSearch)

var prevQuery = ""
async function doSearch() {
    const query = searchInput.value
    if (query === prevQuery) {
        return
    }
    prevQuery = query

    if (query.length === 0) {
        searchResults.style.display = 'none'
        return
    }

    const results = filterItems(query, await searchItems)

    const group = searchResults.getElementsByClassName("list-group")[0]
    group.innerHTML = ''
    
    for (const r of results) {
        const container = document.createElement('li')
        container.classList.add('list-group-item')
        container.classList.add('result')

        const link = document.createElement('a')
        link.setAttribute("href", r.item.uri)
        link.setAttribute("title", "score: " + JSON.stringify(r.score))

        let prev = 0
        for (const [begin, end] of r.indices) {
            if (begin > prev) {
                const text = r.item.name.substring(prev, begin)
                link.append(text)
            }

            const elem = document.createElement('mark')
            elem.innerText = r.item.name.substring(begin, end + 1)
            link.appendChild(elem)
            prev = end + 1
        }

        if (r.item.name.length > prev) {
            const text = r.item.name.substr(prev)
            link.append(text)
        }

        container.appendChild(link)
        container.append(' ')

        const type = document.createElement('span')
        const sticky = document.createElement('span')
        sticky.classList.add('sticky-note')

        switch (r.item.type) {
            case "command":
            case "event":
            case "timeout":
                type.classList.add('role-' + r.item.type)
                type.appendChild(sticky)
                type.append(" ")
                break;

            case "aggregate":
            case "process":
            case "projection":
            case "integration":
                type.classList.add('handlertype-' + r.item.type)
                type.appendChild(sticky)
                type.append(" ")
                break;
        }

        type.append(r.item.type)
        container.appendChild(type)

        if (r.item.docs) {
            const docs = document.createElement('div')
            docs.classList.add('docs')
            
            const text = document.createTextNode(r.item.docs)
            docs.appendChild(text)

            container.appendChild(docs)
        }

        group.appendChild(container)
    }

    if (!group.innerHTML) {
        const message = document.createElement('li')
        message.classList.add('list-group-item')
        message.classList.add('text-muted')
        message.innerText = "No results match this query."
        group.appendChild(message)
    }

    searchResults.style.display = 'block'
}

searchInput.addEventListener('keydown', (event) => {
    if (searchResults.style.display !== 'block') {
        return
    }

    if (event.altKey || event.ctrlKey || event.shiftKey || event.metaKey) {
        return
    }

    const group = searchResults.getElementsByClassName("list-group")[0]
    const active = group.getElementsByClassName('active')[0]

    switch (event.key) {
        case "Escape":
            event.preventDefault()
            searchInput.value = '';
            prevQuery = '';
            searchResults.style.display = 'none';
            break;

        case "ArrowUp":
            if (active) {
                const prev = active.previousSibling
                active.classList.remove('active')

                if (prev) {
                    prev.classList.add('active')
                    prev.scrollIntoViewIfNeeded(true)
                }
            }
            break;
        
        case "ArrowDown":
            if (active) {
                const next = active.nextSibling

                if (next) {
                    active.classList.remove('active')
                    next.classList.add('active')
                    next.scrollIntoViewIfNeeded(false)
                }
            } else {
                const next = group.firstChild
                if (next && next.classList.contains('result')) {
                    next.classList.add('active') 
                    next.scrollIntoViewIfNeeded(false)
                }
            }
            break;
    }

})

document.addEventListener("keyup", (event) => {
    if (event.altKey || event.ctrlKey || event.shiftKey || event.metaKey) {
        return
    }

    if (event.key !== "/" && event.key !== "s") return
    if (/^(?:input|textarea|select|button)$/i.test(event.target.tagName)) return

    event.preventDefault();
    searchInput.focus();
});