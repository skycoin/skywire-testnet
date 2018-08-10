import {FormControl, ValidationErrors} from "@angular/forms";
import StringUtils from "../utils/stringUtils";

function isValidPublicKey(value: string, required: boolean)
{
  if (value !== null && value !== undefined)
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
        return invalid();
      }
    }
  }
  return correct();
}

function publicKeyValidator(required: boolean = false)
{
  return (control: FormControl) =>
  {
    const value = control.value;
    return isValidPublicKey (value, required);
  }
}

function domainValidator(control: FormControl): ValidationErrors
{
  const value = control.value;

  if (value && value.length > 0)
  {
    const host = value.split(':');

    if (host.length !== 2)
    {
      return invalid();
    }

    const port = parseInt(host[1], 10);

    if (isNaN(port) || port <= 0 || port > 65535)
    {
      return invalid();
    }
  }

  return correct();
}

/*** UTIL FUNCTIONS **/

function correct()
{
  return null;
}

function invalid()
{
  return { invalid: true };
}

export {publicKeyValidator, domainValidator};
