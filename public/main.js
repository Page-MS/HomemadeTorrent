const UPDATE_INTERVAL = 1000;
export async function fetchInfo() {
    const container = document.getElementById('register-info');
    await fetchInfoImpl(container);

    setInterval(async () => {
        await fetchInfoImpl(container);
        console.log("register updated");
    }, UPDATE_INTERVAL)
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
                        </li>
                    `;
        });
        html += '</ul>';
        container.innerHTML = html;
    } catch (err) {
        console.error('Failed to fetch info:', err);
        container.innerHTML = '<p class="italic text-red-600">Failed to load.</p>';
    }
}