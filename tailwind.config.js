let colors = require("tailwindcss/colors")
let dark = require("./css/dark");

module.exports = {
  content: ["./data/html/*.html", "./build/data/html/*.html", "./ts/*.ts", "./ts/modules/*.ts"],
  darkMode: 'class',
  theme: {
      extend: {
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
      },
  },
  plugins: [require("a17t")],
}
