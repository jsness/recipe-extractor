import { Recipe } from "../types";

const escapeHTML = (value: string) => value
  .replace(/&/g, "&amp;")
  .replace(/</g, "&lt;")
  .replace(/>/g, "&gt;")
  .replace(/"/g, "&quot;")
  .replace(/'/g, "&#39;");

const buildPrintableRecipeHTML = (recipe: Recipe) => {
  const timeEntries = recipe.times ? Object.entries(recipe.times) : [];

  const metadataHTML = [
    recipe.yield ? `<p><strong>Yield:</strong> ${escapeHTML(recipe.yield)}</p>` : "",
    ...timeEntries.filter(([, value]) => value).map(([key, value]) => (
      `<p><strong>${escapeHTML(key)}:</strong> ${escapeHTML(value)}</p>`
    )),
    `<p><strong>Source:</strong> ${escapeHTML(recipe.source_url)}</p>`,
  ].filter(Boolean).join("");

  const ingredientsHTML = recipe.ingredients.map((group) => `
    <section>
      ${group.group ? `<h3>${escapeHTML(group.group)}</h3>` : ""}
      <ul>
        ${group.items.map((item) => `<li>${escapeHTML(item)}</li>`).join("")}
      </ul>
    </section>
  `).join("");

  const instructionsHTML = recipe.instructions
    .map((step) => `<li>${escapeHTML(step)}</li>`)
    .join("");

  const notesHTML = recipe.notes
    ? `<section><h2>Notes</h2><p>${escapeHTML(recipe.notes)}</p></section>`
    : "";

  return `
    <!DOCTYPE html>
    <html lang="en">
      <head>
        <meta charset="utf-8" />
        <title>${escapeHTML(recipe.title)}</title>
        <style>
          body {
            font-family: "Open Sans", Arial, sans-serif;
            color: #111;
            margin: 2rem auto;
            max-width: 760px;
            line-height: 1.5;
            padding: 0 1.25rem 3rem;
          }
          h1, h2, h3 {
            margin-bottom: 0.5rem;
          }
          h1 {
            font-size: 2rem;
          }
          h2 {
            margin-top: 2rem;
            font-size: 1.2rem;
            border-bottom: 1px solid #ddd;
            padding-bottom: 0.25rem;
          }
          h3 {
            font-size: 1rem;
            margin-top: 1rem;
          }
          p, li {
            font-size: 0.95rem;
          }
          ul, ol {
            padding-left: 1.25rem;
          }
          section + section {
            margin-top: 1rem;
          }
        </style>
      </head>
      <body>
        <h1>${escapeHTML(recipe.title)}</h1>
        ${metadataHTML}
        <section>
          <h2>Ingredients</h2>
          ${ingredientsHTML}
        </section>
        <section>
          <h2>Instructions</h2>
          <ol>${instructionsHTML}</ol>
        </section>
        ${notesHTML}
      </body>
    </html>
  `;
};

export const printRecipe = (recipe: Recipe) => {
  const iframe = document.createElement("iframe");
  const printableHTML = buildPrintableRecipeHTML(recipe);
  iframe.style.position = "fixed";
  iframe.style.right = "0";
  iframe.style.bottom = "0";
  iframe.style.width = "0";
  iframe.style.height = "0";
  iframe.style.border = "0";
  iframe.setAttribute("aria-hidden", "true");

  const cleanup = () => {
    window.setTimeout(() => {
      iframe.remove();
    }, 1000);
  };

  iframe.onload = () => {
    const printWindow = iframe.contentWindow;
    if (!printWindow) {
      cleanup();
      return;
    }

    printWindow.focus();
    printWindow.print();
    cleanup();
  };

  iframe.srcdoc = printableHTML;
  document.body.appendChild(iframe);
};
