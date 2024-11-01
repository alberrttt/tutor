// generate-fonts.js

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

/**
 * Vite Plugin to Generate @font-face Declarations
 */
export default function generateFontsPlugin() {
    return {
        name: 'vite-plugin-generate-fonts',
        buildStart() {
            generateFontCSS();
        },
        configureServer() {
            // Optionally regenerate fonts during development when the server starts
            generateFontCSS();
        },
    };
}

/**
 * Function to Generate fonts.tmp.css with @font-face Declarations
 */
function generateFontCSS() {
    // Recreate __dirname in ES modules
    const __filename = fileURLToPath(import.meta.url);
    const __dirname = path.dirname(__filename);

    // Define paths
    const fontDir = path.join(__dirname, 'static', 'nunito');
    const tmpCssPath = path.join(__dirname, 'src', 'fonts.tmp.css');

    // Check if font directory exists
    if (!fs.existsSync(fontDir)) {
        console.error(`Font directory not found: ${fontDir}`);
        return;
    }

    // Read all .ttf files in the font directory
    const fontFiles = fs.readdirSync(fontDir).filter(file => file.endsWith('.ttf'));

    if (fontFiles.length === 0) {
        console.warn(`No .ttf files found in ${fontDir}`);
        return;
    }

    // Initialize CSS content
    let cssContent = `/* Auto-generated font definitions */\n\n`;

    fontFiles.forEach(file => {
        const fontName = 'Nunito';
        const fileName = path.basename(file, '.ttf'); // e.g., Nunito-BoldItalic
        const variant = fileName.split('-')[1] || 'Regular'; // Extract variant

        // Determine font-weight and font-style based on variant
        let fontWeight = 'normal';
        let fontStyle = 'normal';

        if (/bold/i.test(variant)) fontWeight = 'bold';
        if (/italic/i.test(variant)) fontStyle = 'italic';

        cssContent += `
@font-face {
  font-family: '${fontName}';
  src: url('/nunito/${file}') format('truetype');
  font-weight: ${fontWeight};
  font-style: ${fontStyle};
  font-display: swap;
}
`;
    });

    // Write the CSS content to the temporary file
    try {
        fs.writeFileSync(tmpCssPath, cssContent, 'utf8');
        console.log(`Fonts have been generated in ${tmpCssPath}`);
    } catch (error) {
        console.error(`Failed to write to ${tmpCssPath}:`, error);
    }
}