import { max } from "d3-array";

export class Aligner {
  private pre: string[] = [];
  private post: string[] = [];

  public add(pre: string, post: string): Aligner {
    this.pre.push(pre);
    this.post.push(post);
    return this;
  }

  public toString(): string {
    // determine the maximum length of the pre-strings
    const maxPreLength = max(this.pre, d => d.length) ?? 0;
    let ret = "";
    for (let i = 0; i < this.pre.length; i++) {
      ret += `${this.pre[i].padEnd(maxPreLength + 1) + this.post[i]}\n`;
    }
    ret = ret.trimEnd();
    return ret;
  }
}
