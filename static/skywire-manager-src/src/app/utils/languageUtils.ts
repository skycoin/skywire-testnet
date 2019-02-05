const LANGUAGES_LIST = {
  en: {
    name: 'English',
    nativeName: 'English',
  },
  // es: {
  //   name: 'Spanish',
  //   nativeName: 'Español',
  // }
};

function getNativeName(langCode: string): string {
  return LANGUAGES_LIST[langCode].nativeName;
}

function getLangs(): string[] {
  return Object.keys(LANGUAGES_LIST);
}

export {
  getNativeName,
  getLangs
};
