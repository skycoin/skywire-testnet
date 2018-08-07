export default class StringUtils
{
  /**
   * Removes whitespaces from a string
   * @param {string} value
   * @returns {string} the same string without whitespaces
   */
  static removeWhitespaces(value: string): string
  {
    return value.replace(/\s/g,'');
  }
}
