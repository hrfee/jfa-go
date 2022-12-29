let colors = require("tailwindcss/colors")
let dark = require("../css/dark");

module.exports = {
  content: ["./safelist.html", "./index.html", "./out/main.js"],
  darkMode: 'class',
  important: true,
  theme: {
      extend: {
          keyframes: {
              'fade-in': {
                  '0%': {
                      opacity: '0'
                  },
                  '100%': {
                      opacity: '1'
                  }
              },
              'fade-out': {
                  '0%': {
                      opacity: '1'
                  },
                  '100%': {
                      opacity: '0'
                  }
              },
          },
          animation: {
              'fade-in': 'fade-in 0.2s cubic-bezier(0.25, 0.46, 0.45, 0.94)',
              'fade-out': 'fade-out 0.2s cubic-bezier(0.25, 0.46, 0.45, 0.94)'
          },
          colors: {
              neutral: colors.slate,
              positive: colors.green,
              urge: colors.violet,
              warning: colors.yellow,
              info: colors.blue,
              critical: colors.red,
              d_neutral: dark.d_neutral,
              d_positive: dark.d_positive,
              d_urge: dark.d_urge,
              d_warning: dark.d_warning,
              d_info: dark.d_info,
              d_critical: dark.d_critical
          }
      }
  },
  safelist: ["block", "animate-fade-in", "animate-fade-out"],
  plugins: [require("a17t")],
}
