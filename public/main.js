const UPDATE_INTERVAL = 100000;
export async function fetchInfo() {
    const container = document.getElementById('register-info');
    await fetchInfoImpl(container);

    setInterval(async () => {
        await fetchInfoImpl(container);
        console.log("register updated");
    }, UPDATE_INTERVAL)
}

function updateStatus(msg, isError = false) {
    const statusEl = document.getElementById('status');
    statusEl.textContent = msg;

    if (isError) {
        statusEl.className = 'block mt-3 p-2 border border-red-600 bg-red-50 text-red-700 font-bold text-sm';
    } else {
        statusEl.className = 'block mt-3 p-2 border border-green-600 bg-green-50 text-green-700 font-bold text-sm';
    }

    setTimeout(() => {
        statusEl.className = 'hidden';
    }, 3000);
}

const msgInput = document.getElementById('message');
export async function sendMessage(msg) {
    if (!msg) return;

    try {
        const formData = new URLSearchParams();
        formData.append('message', msg);

        const response = await fetch('/api/send', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: formData
        });

        console.log(response);
        if (response.ok) {
            updateStatus('Message sent successfully!');
            msgInput.value = '';
        } else {
            updateStatus('Failed to send message: ' + response.statusText, true);
        }
    } catch (err) {
        console.error(err);
        updateStatus('Network error while sending', true);
    }
}

async function fetchInfoImpl(container) {
    try {
        const response = await fetch('/api/info');
        const data = await response.json();
        if (!data.Files || data.Files.length === 0) {
            container.innerHTML = '<p class="italic text-stone-600">No files in register.</p>';
            return;
        }

        let html = '<ul class="m-0 p-0 list-none space-y-2">';
        data.Files.forEach(file => {
            const { id, name, file_parts, peers_with_file, size } = file;
            const hasFile = peers_with_file && peers_with_file.includes(data.SiteID);
            const statusClass = hasFile ? 'text-green-700 font-bold' : 'text-stone-500';
            const statusText = hasFile ? '[Available Locally]' : '[Available on network]';

            html += `
                        <li class="border border-stone-200 bg-white p-3">
                            <div class="flex justify-between items-start">
                                <span class="font-bold">${name}</span>
                                <span class="text-sm ${statusClass}">${statusText}</span>
                            </div>
                            <div class="text-sm text-elispis text-stone-600 mt-2 pt-2 border-t border-dotted border-stone-200">
                                ID: ${id.slice(0, 10) || "N/A"}<br>
                                Parts: ${file_parts.length}
                                ${peers_with_file ? '<br>Peers: ' + peers_with_file.join(', ') : ''}
                            </div>
                            ${!hasFile ? peers_with_file.slice(0, 1).map(p => `<button
data-message="${getDownloadPayload(id, p)}"
class="download mt-2 self-start bg-stone-200
border border-stone-500 py-.5 px-1 text-stone-800 hover:bg-stone-300 transition-colors">
Télécharger</button>`) : ''}
                        </li>
                    `;
        });
        html += '</ul>';
        container.innerHTML = html;

        document.querySelectorAll('.download').forEach(el => {
            el.addEventListener('click', async (e) => {
                const msg = e.target.dataset.message;
                sendMessage(msg);
            })
        });
    } catch (err) {
        console.error('Failed to fetch info:', err);
        container.innerHTML = '<p class="italic text-red-600">Failed to load.</p>';
    }
}

function getDownloadPayload(fileId, dest) {
    const ACTION      = `ACTION:StartTransfers
`;
    //const DEST        = `DEST:${dest}
    const DEST        = `DEST:${globalThis.siteInfo.siteId}
`;
    const PAYLOAD     = `FileID;${fileId}
`;
    const PAYLOAD_LEN = `PAYLOAD_LEN:${PAYLOAD.length}
`;

    return `${ACTION}${DEST}${PAYLOAD_LEN}${PAYLOAD}

`
}