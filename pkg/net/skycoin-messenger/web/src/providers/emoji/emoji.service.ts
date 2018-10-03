import { Injectable } from '@angular/core';
import * as emojione from 'emojione';

@Injectable()
export class EmojiService {
  baseEmojis = [
    { code: `<img class="emojione" alt="😀" title=":grinning:" src="/emojis/1f600.png"/>`, short: ':grinning:', desc: 'Grinning face' },
    { code: `<img class="emojione" alt="😁" title=":grin:" src="/emojis/1f601.png"/>`, short: ':grin:', desc: 'Grinning face with smiling eyes' },
    { code: `<img class="emojione" alt="😂" title=":joy:" src="/emojis/1f602.png"/>`, short: ':joy:', desc: 'Face with tears of joy' },
    { code: `<img class="emojione" alt="😃" title=":smiley:" src="/emojis/1f603.png"/>`, short: ':smiley:', desc: 'Smiling face with open mouth' },
    { code: `<img class="emojione" alt="😄" title=":smile:" src="/emojis/1f604.png"/>`, short: ':smile:', desc: 'Smiling face with open mouth and smiling eyes' },
    { code: `<img class="emojione" alt="😅" title=":sweat_smile:" src="/emojis/1f605.png"/>`, short: ':sweat_smile:', desc: 'Smiling face with open mouth and cold sweat' },
    { code: `<img class="emojione" alt="😆" title=":laughing:" src="/emojis/1f606.png"/>`, short: ':laughing:', desc: 'Smiling face with open mouth and tightly-closed eyes' },
    { code: `<img class="emojione" alt="😇" title=":innocent:" src="/emojis/1f607.png"/>`, short: ':innocent:', desc: 'Smiling face with halo' },

    { code: `<img class="emojione" alt="😈" title=":smiling_imp:" src="/emojis/1f608.png"/>`, short: ':smiling_imp:', desc: 'Smiling face with horns' },
    { code: `<img class="emojione" alt="😉" title=":wink:" src="/emojis/1f609.png"/>`, short: ':wink:', desc: 'Winking face' },
    { code: `<img class="emojione" alt="😊" title=":blush:" src="/emojis/1f60a.png"/>`, short: ':blush:', desc: 'Smiling face with smiling eyes' },
    { code: `<img class="emojione" alt="😋" title=":yum:" src="/emojis/1f60b.png"/>`, short: ':yum:', desc: 'Face savoring delicious food' },
    { code: `<img class="emojione" alt="😌" title=":relieved:" src="/emojis/1f60c.png"/>`, short: ':relieved:', desc: 'Relieved face' },
    { code: `<img class="emojione" alt="😍" title=":heart_eyes:" src="/emojis/1f60d.png"/>`, short: ':heart_eyes:', desc: 'Smiling face with heart-shaped eyes' },
    { code: `<img class="emojione" alt="😎" title=":sunglasses:" src="/emojis/1f60e.png"/>`, short: ':sunglasses:', desc: 'Smiling face with sunglasses' },
    { code: `<img class="emojione" alt="😏" title=":smirk:" src="/emojis/1f60f.png"/>`, short: ':smirk:', desc: 'Smirking face' },

    { code: `<img class="emojione" alt="😐" title=":neutral_face:" src="/emojis/1f610.png"/>`, short: ':neutral_face:', desc: 'Neutral face' },
    { code: `<img class="emojione" alt="😑" title=":expressionless:" src="/emojis/1f611.png"/>`, short: ':expressionless:', desc: 'Expressionless face' },
    { code: `<img class="emojione" alt="😒" title=":unamused:" src="/emojis/1f612.png"/>`, short: ':unamused:', desc: 'Unamused face' },
    { code: `<img class="emojione" alt="😓" title=":sweat:" src="/emojis/1f613.png"/>`, short: ':sweat:', desc: 'Face with cold sweat' },
    { code: `<img class="emojione" alt="😔" title=":pensive:" src="/emojis/1f614.png"/>`, short: ':pensive:', desc: 'Pensive face' },
    { code: `<img class="emojione" alt="😕" title=":confused:" src="/emojis/1f615.png"/>`, short: ':confused:', desc: 'Confused face' },
    { code: `<img class="emojione" alt="😖" title=":confounded:" src="/emojis/1f616.png"/>`, short: ':confounded:', desc: 'Confounded face' },
    { code: `<img class="emojione" alt="😗" title=":kissing:" src="/emojis/1f617.png"/>`, short: ':kissing:', desc: 'Kissing face' },

    { code: `<img class="emojione" alt="😘" title=":kissing_heart:" src="/emojis/1f618.png"/>`, short: ':kissing_heart:', desc: 'Face throwing a kiss' },
    { code: `<img class="emojione" alt="😙" title=":kissing_smiling_eyes:" src="/emojis/1f619.png"/>`, short: ':kissing_smiling_eyes:', desc: 'Kissing face with smiling eyes' },
    { code: `<img class="emojione" alt="😚" title=":kissing_closed_eyes:" src="/emojis/1f61a.png"/>`, short: ':kissing_closed_eyes:', desc: 'Kissing face with closed eyes' },
    { code: `<img class="emojione" alt="😛" title=":stuck_out_tongue:" src="/emojis/1f61b.png"/>`, short: ':stuck_out_tongue:', desc: 'Face with stuck out tongue' },
    { code: `<img class="emojione" alt="😜" title=":stuck_out_tongue_winking_eye:" src="/emojis/1f61c.png"/>`, short: ':stuck_out_tongue_winking_eye:', desc: 'Face with stuck out tongue and winking eye' },
    { code: `<img class="emojione" alt="😝" title=":stuck_out_tongue_closed_eyes:" src="/emojis/1f61d.png"/>`, short: ':stuck_out_tongue_closed_eyes:', desc: 'Face with stuck out tongue and tightly-closed eyes' },
    { code: `<img class="emojione" alt="😞" title=":disappointed:" src="/emojis/1f61e.png"/>`, short: ':disappointed:', desc: 'Disappointed face' },
    { code: `<img class="emojione" alt="😟" title=":worried:" src="/emojis/1f61f.png"/>`, short: ':worried:', desc: 'Worried face' },

    { code: `<img class="emojione" alt="😠" title=":angry:" src="/emojis/1f620.png"/>`, short: ':angry:', desc: 'Angry face' },
    { code: `<img class="emojione" alt="😡" title=":rage:" src="/emojis/1f621.png"/>`, short: ':rage:', desc: 'Pouting face' },
    { code: `<img class="emojione" alt="😢" title=":cry:" src="/emojis/1f622.png"/>`, short: ':cry:', desc: 'Crying face' },
    { code: `<img class="emojione" alt="😣" title=":persevere:" src="/emojis/1f623.png"/>`, short: ':persevere:', desc: 'Persevering face' },
    { code: `<img class="emojione" alt="😤" title=":triumph:" src="/emojis/1f624.png"/>`, short: ':triumph:', desc: 'Face with look of triumph' },
    { code: `<img class="emojione" alt="😥" title=":disappointed_relieved:" src="/emojis/1f625.png"/>`, short: ':disappointed_relieved:', desc: 'Disappointed but relieved face' },
    { code: `<img class="emojione" alt="😦" title=":frowning:" src="/emojis/1f626.png"/>`, short: ':frowning:', desc: 'Frowning face with open mouth' },
    { code: `<img class="emojione" alt="😧" title=":anguished:" src="/emojis/1f627.png"/>`, short: ':anguished:', desc: 'Anguished face' },

    { code: `<img class="emojione" alt="😨" title=":fearful:" src="/emojis/1f628.png"/>`, short: ':fearful:', desc: 'Fearful face' },
    { code: `<img class="emojione" alt="😩" title=":weary:" src="/emojis/1f629.png"/>`, short: ':weary:', desc: 'Weary face' },
    { code: `<img class="emojione" alt="😪" title=":sleepy:" src="/emojis/1f62a.png"/>`, short: ':sleepy:', desc: 'Sleepy face' },
    { code: `<img class="emojione" alt="😫" title=":tired_face:" src="/emojis/1f62b.png"/>`, short: ':tired_face:', desc: 'Tired face' },
    { code: `<img class="emojione" alt="😬" title=":grimacing:" src="/emojis/1f62c.png"/>`, short: ':grimacing:', desc: 'Grimacing face' },
    { code: `<img class="emojione" alt="😭" title=":sob:" src="/emojis/1f62d.png"/>`, short: ':sob:', desc: 'Loudly crying face' },
    { code: `<img class="emojione" alt="😮" title=":open_mouth:" src="/emojis/1f62e.png"/>`, short: ':open_mouth:', desc: 'Face with open mouth' },
    { code: `<img class="emojione" alt="😯" title=":hushed:" src="/emojis/1f62f.png"/>`, short: ':hushed:', desc: 'Hushed face' },

    { code: `<img class="emojione" alt="😰" title=":cold_sweat:" src="/emojis/1f630.png"/>`, short: ':cold_sweat:', desc: 'Face with open mouth and cold sweat' },
    { code: `<img class="emojione" alt="😱" title=":scream:" src="/emojis/1f631.png"/>`, short: ':scream:', desc: 'Face screaming in fear' },
    { code: `<img class="emojione" alt="😲" title=":astonished:" src="/emojis/1f632.png"/>`, short: ':astonished:', desc: 'Astonished face' },
    { code: `<img class="emojione" alt="😳" title=":flushed:" src="/emojis/1f633.png"/>`, short: ':flushed:', desc: 'Flushed face' },
    { code: `<img class="emojione" alt="😴" title=":sleeping:" src="/emojis/1f634.png"/>`, short: ':sleeping:', desc: 'Sleeping face' },
    { code: `<img class="emojione" alt="😵" title=":dizzy_face:" src="/emojis/1f635.png"/>`, short: ':dizzy_face:', desc: 'Dizzy face' },
    { code: `<img class="emojione" alt="😶" title=":no_mouth:" src="/emojis/1f636.png"/>`, short: ':no_mouth:', desc: 'Face without mouth' },
    { code: `<img class="emojione" alt="😷" title=":mask:" src="/emojis/1f637.png"/>`, short: ':mask:', desc: 'Face with medical mask' }]
  constructor() { }

  toImage(shorName: string) {
    emojione.imagePathPNG = '/emojis/'
    return emojione.shortnameToImage(shorName);
  }

  getPeopleList() {
    const list: Map<string, EmojiData> = emojione.emojioneList;
    const map = new Map<string, EmojiData>();
    for (const key in list) {
      if (list.hasOwnProperty(key)) {
        const element: EmojiData = list[key];
        if (element.category === 'people') {
          map.set(key, element);
          // console.log('test:', element);
        }
      }
    }
    return map;
  }
}

export interface EmojiData {
  uc_base?: string;
  uc_output?: string;
  uc_match?: string;
  uc_greedy?: string;
  shortnames?: Array<string>;
  category?: string;
}
