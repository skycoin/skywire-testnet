import {FormControl} from "@angular/forms";
import StringUtils from "../utils/stringUtils";

function isValidPublicKey(value: string, required: boolean)
{
  if (value)
  {
    if (required || value.length > 0)
    {
      const isEmpty = StringUtils.removeWhitespaces(value).length === 0,
            isValid = (value as string).length === 66;

      if (isEmpty)
      {
        return { required: true }
      }
      else if (!isValid)
      {
        return { invalid: true }
      }
    }
  }
  return null;
}

function publicKey (required: boolean = false)
{
  return (control: FormControl) =>
  {
    const value = control.value;
    return isValidPublicKey (value, required);
  }
}

export default publicKey;
