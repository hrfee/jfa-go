let colors = require("tailwindcss/colors")
let dark = require("./css/dark");

module.exports = {
  content: ["./data/html/*.html", "./build/data/html/*.html", "./ts/*.ts", "./ts/modules/*.ts"],
  darkMode: 'class',
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
              'slide-in': {
                  '0%': {
                      opacity: '0',
                      transform: 'translateY(-100%)'
                  },
                  '100%': {
                      opacity: '1',
                      transform: 'translateY(0%)'
                  },
              },
              'slide-out': {
                  '0%': {
                      opacity: '1',
                      transform: 'translateY(0%)'
                  },
                  '100%': {
                      opacity: '0',
                      transform: 'translateY(-100%)'
                  },
              },
              'pulse': {
                  '0%': {
                      transform: 'scale(1)'
                  },
                  '50%': {
                      transform: 'scale(1.05)'
                  },
                  '100%': {
                      transform: 'scale(1)'
                  }
              }
          },
          animation: {
              'fade-in': 'fade-in 0.2s cubic-bezier(0.25, 0.46, 0.45, 0.94)',
              'fade-out': 'fade-out 0.2s cubic-bezier(0.25, 0.46, 0.45, 0.94)',
              'slide-in': 'slide-in 0.2s cubic-bezier(.08,.52,.01,.98)',
              'slide-out': 'slide-out 0.2s cubic-bezier(.08,.52,.01,.98)',
              'pulse': 'pulse 0.2s cubic-bezier(0.25, 0.45, 0.45, 0.94)'
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
              d_critical: dark.d_critical,
              discord: "#5865F2"
          }
      }
  },
  plugins: [require("a17t")],
}
