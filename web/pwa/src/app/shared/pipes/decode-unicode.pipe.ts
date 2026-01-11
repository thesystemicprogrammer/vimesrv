import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'decodeUnicode',
  standalone: true
})
export class DecodeUnicodePipe implements PipeTransform {
  transform(value: string | null | undefined): string {
    if (!value) return '';
    return value.replace(/\\u[\dA-Fa-f]{4}/g, m =>
      String.fromCharCode(parseInt(m.slice(2), 16))
    );
  }
}
