export interface SkillFrontmatter {
  name: string;
  version?: string;
  description?: string;
  tags?: string[];
}

export interface Skill {
  name: string;
  content: string;
  body: string;
  frontmatter: SkillFrontmatter;
  filePath: string;
  sizeBytes: number;
}
