/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: "#0b141a",
        surface: "#111b21",
        "surface-2": "#182229",
        border: "#2a3942",
        muted: "#8696a0",
        accent: "#00a884",
        green: "#25d366",
        yellow: "#ffb703",
        red: "#f04747",
        "bubble-in": "#202c33",
        "bubble-out": "#005c4b",
      },
    },
  },
  plugins: [],
};
