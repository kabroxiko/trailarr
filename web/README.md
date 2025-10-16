# React + Vite

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Babel](https://babeljs.io/) for Fast Refresh
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/) for Fast Refresh

## Expanding the ESLint configuration

If you are developing a production application, we recommend using TypeScript with type-aware lint rules enabled. Check out the [TS template](https://github.com/vitejs/vite/tree/main/packages/create-vite/template-react-ts) for information on how to integrate TypeScript and [`typescript-eslint`](https://typescript-eslint.io) in your project.

Testing the Alternate Titles hover
---------------------------------

When a media item contains alternate titles (field `alternateTitles`, `alternate_titles`, or `AlternateTitles`), the Media Details view shows a small language icon next to the title. Hover the icon or focus it with keyboard to see a tooltip listing alternate titles.

Quick manual check:

- Start the app and open a Media Details page for an item that has alternate titles.
- Move the mouse over the small language icon next to the title (or tab to it) and the tooltip list should appear.
