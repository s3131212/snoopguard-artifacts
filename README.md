# Bots can Snoop: Uncovering and Mitigating Privacy Risks of Bots in Group Chats

**Authors**: Kai-Hsiang Chou, Yi-Min Lin, Yi-An Wang, Jonathan Weiping Li, Tiffany Hyun-Jin Kim, Hsu-Chun Hsiao

This is the repository for the artifact accompanying the paper "Bots Can Snoop: Uncovering and Mitigating Privacy Risks of Bots in Group Chats."

**Full Paper**: [arXiv:2410.06587](https://arxiv.org/abs/2410.06587)

## Repository Structure

- **`analyze_pushshift/`** contains data processing and analysis scripts for the Pushshift Telegram Dataset. These scripts analyze the prevalence of users encountering the same chatbots across different groups. Detailed discussions and results are in Section 3.2 of the paper.

- **`chatbot-testing-scripts/`** contains basic chatbot implementations for popular platforms, including Discord, Keybase, LINE, Slack, Telegram, and WhatsApp. These implementations help investigate the extent of information accessible to chatbots on each platform. Full details are available in Section 4 of the paper.

- **`snoopguard-poc/`** includes the reference implementation of **SnoopGuard**, a secure group messaging protocol designed to protect user privacy from chatbots while maintaining strong end-to-end encryption. The introduction and benchmarking of this implementation are discussed in Section 6.2.2 of the paper.

Each directory contains its own `README` file with additional details about its content and usage.


## Citation

If you use our work, please cite our paper:

```bibtex
@article{chou2024bots,
  title={Bots can Snoop: Uncovering and Mitigating Privacy Risks of Bots in Group Chats},
  author={Chou, Kai-Hsiang and Lin, Yi-Min and Wang, Yi-An and Li, Jonathan Weiping and Kim, Tiffany Hyun-Jin and Hsiao, Hsu-Chun},
  journal={arXiv preprint arXiv:2410.06587},
  year={2024}
}
```