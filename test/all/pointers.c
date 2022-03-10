int f(int y, int z)
{
    int a = 10;
    int *b = &a;
    int **c = &b;
    int ***d = &c;
    int **e = *d;
    int f = 3;
    **e += 2;
    f += 1;
    return f + **e + 5;
}
